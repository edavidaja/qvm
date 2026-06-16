# q2 (Quarto v2) support in qvm

**Date:** 2026-06-16
**Status:** Approved design

## Problem

The Quarto team is developing Quarto v2 ("q2") in a separate repository,
`quarto-dev/q2`, with its own release stream. qvm currently manages only
classic Quarto ("q1") from `quarto-dev/quarto-cli`. Nearly every assumption in
the install/detect/use path differs between the two editions:

| Aspect          | q1 (`quarto`)                                                   | q2 (`q2`)                                          |
| --------------- | --------------------------------------------------------------- | -------------------------------------------------- |
| GitHub repo     | `quarto-dev/quarto-cli`                                          | `quarto-dev/q2`                                    |
| Asset names     | `quarto-{v}-macos.tar.gz`, `-linux-{arch}.tar.gz`, `-win.zip`, rhel7 | `q2-{v}-{os}_{arch}.{tar.gz,zip}`             |
| macOS builds    | single universal                                                | per-arch (amd64 / arm64)                           |
| Archive shape   | directory tree → `bin/quarto`, `share/`, …                      | single binary `q2` at archive root                 |
| Versions dir    | `…/qvm/versions/<v>/bin/quarto`                                  | `…/qvm/versions-q2/<v>/q2`                          |
| Active symlink  | `qvm/bin/quarto`                                                 | `qvm/bin/q2`                                        |
| Checksum verify | none                                                             | yes (`.sha256`)                                    |

q2 currently tags releases as `v0.1.x`; those numbers will eventually overlap
q1's numbering, so editions must be namespaced rather than merged into one
version list.

## Goals

- Manage q2 versions (install / use / ls / rm) alongside q1.
- q1 and q2 are **parallel tracks with distinct names**: q1 stays `quarto` on
  PATH, q2 is `q2`. Both can be active simultaneously; versions are chosen
  independently per edition.
- Edition is selected by a `--q2` flag on existing commands. Default (no flag)
  is q1 and preserves today's behavior exactly.
- Verify q2 downloads with their published `.sha256` checksums.

## Non-goals (YAGNI)

- Per-edition default-version configuration.
- minisign signature verification (q2 also ships `.minisig`).
- Exposing q2 on PATH as `quarto`.
- An `qvm q2 …` subcommand namespace (rejected in favor of the `--q2` flag).
- Auto-detecting edition from the version string (fragile; version ranges
  overlap).

## Design

### Core abstraction: `Edition`

The q1 specifics today are hardcoded literals scattered across `gh`,
`pipeline`, and `config`. A new low-dependency package `internal/edition`
captures everything that differs between editions in one value, and the other
packages take an `Edition` as a parameter instead of hardcoding q1.

```go
// internal/edition/edition.go
package edition

type Kind int

const (
	Q1 Kind = iota // classic quarto (quarto-dev/quarto-cli) — zero value
	Q2             // quarto v2 (quarto-dev/q2)
)

// Edition captures where to fetch an edition, how its release assets are
// named, how the archive is laid out, and where/under what name it lives once
// installed.
type Edition struct {
	Kind Kind

	Owner string // GitHub owner, e.g. "quarto-dev"
	Repo  string // GitHub repo, e.g. "quarto-cli" or "q2"

	// ExeName is the binary's name both inside the install tree and as the
	// active symlink on PATH ("quarto" vs "q2").
	ExeName string

	// VerifyChecksum: when true, fetch the matching ".sha256" asset and verify
	// the tarball hash before extraction.
	VerifyChecksum bool

	// assetMatcher resolves the release-asset name suffix for a platform.
	assetMatcher func(p Platform) (string, error)

	// exeRelPath is the path to the executable inside an extracted version
	// dir: "bin/quarto" (q1) or "q2" (q2).
	exeRelPath string
}

// Platform is the resolved target we are installing for.
type Platform struct {
	OS   string // "darwin" | "linux" | "windows" | "rhel7"
	Arch string // "amd64" | "arm64"
}

func For(k Kind) Edition         // Q2 → q2, else q1
func FromFlag(q2flag bool) Edition // true → q2, else q1
```

The zero value of `Kind` is `Q1`, so any code path that never sets an edition
keeps today's behavior.

Asset matchers hold the naming knowledge currently in `assets.go`'s
`osAssetSuffix`. They return a **suffix**, not the full name, preserving the
existing `strings.HasSuffix` matching contract:

```go
// q1: returns "macos.tar.gz", "linux-amd64.tar.gz", "linux-arm64.tar.gz",
//     "win.zip", "linux-rhel7-amd64.tar.gz"
func q1Asset(p Platform) (string, error)

// q2: returns "darwin_arm64.tar.gz", "linux_amd64.tar.gz", "windows_amd64.zip", …
//     errors on unsupported combos (e.g. windows/arm64)
func q2Asset(p Platform) (string, error)
```

### Per-package changes

**`internal/config` (paths & detection)**

- `GetPathToVersionsDir(e Edition)` — `…/qvm/versions` for q1 (unchanged),
  `…/qvm/versions-q2` for q2. The root-as-`/opt/quarto` special case stays
  q1-only.
- `GetInstalledVersions(e Edition)` — scans the edition's versions dir, detects
  a version when `<v>/<e.exeRelPath>` exists.
- `GetActiveVersion(e Edition)` / `GetPathToActiveExe(e Edition)` — resolve the
  symlink named `e.ExeName` inside the shared `qvm/bin` dir. Active-version
  derivation walks up from the depth implied by `e.exeRelPath` (q1: two levels
  `…/<v>/bin/quarto`; q2: one level `…/<v>/q2`).

**`internal/gh` (fetch & download)**

- `GetRelease` / `GetLatestRelease` / `GetReleases` — take `owner, repo`
  (supplied from the edition) instead of hardcoded literals.
- `DownloadReleaseAsset` — takes the `Edition` + resolved `Platform`, uses
  `e.assetMatcher` to choose the asset, keeps suffix-matching.

**`internal/pipeline` (install + verify)**

- `DownloadReleaseVersion(e Edition, release, platform, progress)` — installs
  into the edition's versions dir.
- When `e.VerifyChecksum`: fetch `<asset>.sha256`, compute SHA256 of the
  downloaded tarball, compare, and abort before extraction on mismatch. The q1
  path skips this.

**`cmd` (flag wiring)**

- Add a `--q2` bool flag to `install`, `use`, `ls`, `rm`; resolve via
  `edition.FromFlag(...)` and thread the edition through. No `--q2` → q1,
  identical to today.
- `use` symlinks `e.ExeName` into `qvm/bin`. That dir is already on PATH via
  `init`, so **no shell-init change is needed** — `q2` appears alongside
  `quarto`.

### Error handling

- Unsupported platform/arch (e.g. q2 on windows/arm64) → the asset matcher
  returns a clear error such as `"q2 has no build for windows/arm64"`,
  consistent with the existing improved-error-message work.

### Testing

- Table-driven `assetMatcher` tests per edition × platform, including
  unsupported combinations.
- Checksum verification: happy-path and mismatch (abort-before-extract).
- `GetInstalledVersions` against temp dirs laid out in each edition's shape.
- Reuse the existing `gh` integration-test pattern for live-release checks.

## Open questions

None outstanding.

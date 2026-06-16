// Package edition captures everything that differs between Quarto editions:
// where to fetch them, how their release assets are named, how their archives
// are laid out, and where/under what name each lives once installed.
package edition

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Kind identifies a Quarto edition. The zero value is Q1 so code paths that
// never set an edition keep classic-Quarto behavior.
type Kind int

const (
	Q1 Kind = iota // classic quarto (quarto-dev/quarto-cli)
	Q2             // quarto v2 (quarto-dev/q2)
)

// Platform is the resolved target we are installing for.
type Platform struct {
	OS   string // "darwin" | "linux" | "windows" | "rhel7"
	Arch string // "amd64" | "arm64"
}

// Edition captures where to fetch an edition, how its assets are named, and
// where/under what name it lives once installed.
type Edition struct {
	Kind            Kind
	Owner           string // GitHub owner, e.g. "quarto-dev"
	Repo            string // GitHub repo, e.g. "quarto-cli" or "q2"
	VersionsDirName string // dir under qvm data home, "versions" | "versions-q2"
	VerifyChecksum  bool   // verify the ".sha256" asset before extraction
}

// For returns the Edition for a Kind.
func For(k Kind) Edition {
	if k == Q2 {
		return Edition{
			Kind:            Q2,
			Owner:           "quarto-dev",
			Repo:            "q2",
			VersionsDirName: "versions-q2",
			VerifyChecksum:  true,
		}
	}
	return Edition{
		Kind:            Q1,
		Owner:           "quarto-dev",
		Repo:            "quarto-cli",
		VersionsDirName: "versions",
		VerifyChecksum:  false,
	}
}

// FromFlag maps the --q2 boolean to an edition.
func FromFlag(q2 bool) Edition {
	if q2 {
		return For(Q2)
	}
	return For(Q1)
}

// ExeName is the binary's name on PATH (the active symlink), OS-aware.
func (e Edition) ExeName() string {
	if e.Kind == Q2 {
		if runtime.GOOS == "windows" {
			return "q2.exe"
		}
		return "q2"
	}
	if runtime.GOOS == "windows" {
		return "quarto.cmd"
	}
	return "quarto"
}

// ExeRelPath is the executable's path inside an extracted version dir, OS-aware.
// q1 ships a directory tree (bin/quarto); q2 ships a single binary at the root.
func (e Edition) ExeRelPath() string {
	if e.Kind == Q2 {
		return e.ExeName() // "q2" / "q2.exe"
	}
	return filepath.Join("bin", e.ExeName()) // "bin/quarto" / "bin/quarto.cmd"
}

// AssetSuffix resolves the release-asset filename suffix for a platform.
// Callers match release assets by this suffix (strings.HasSuffix).
func (e Edition) AssetSuffix(p Platform) (string, error) {
	if e.Kind == Q2 {
		return q2AssetSuffix(p)
	}
	return q1AssetSuffix(p)
}

func q1AssetSuffix(p Platform) (string, error) {
	switch p.OS {
	case "darwin":
		return "macos.tar.gz", nil // universal build
	case "linux":
		if p.Arch == "arm64" {
			return "linux-arm64.tar.gz", nil
		}
		return "linux-amd64.tar.gz", nil
	case "windows":
		return "win.zip", nil
	case "rhel7":
		return "linux-rhel7-amd64.tar.gz", nil
	}
	return "", fmt.Errorf("quarto has no build for %s/%s", p.OS, p.Arch)
}

func q2AssetSuffix(p Platform) (string, error) {
	switch p.OS {
	case "darwin", "linux":
		if p.Arch != "amd64" && p.Arch != "arm64" {
			return "", fmt.Errorf("q2 has no build for %s/%s", p.OS, p.Arch)
		}
		return fmt.Sprintf("%s_%s.tar.gz", p.OS, p.Arch), nil
	case "windows":
		if p.Arch != "amd64" {
			return "", fmt.Errorf("q2 has no build for %s/%s", p.OS, p.Arch)
		}
		return "windows_amd64.zip", nil
	}
	return "", fmt.Errorf("q2 has no build for %s/%s", p.OS, p.Arch)
}

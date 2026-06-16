package gh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dpastoor/qvm/internal/edition"
	"github.com/dustin/go-humanize"
	"github.com/google/go-github/v73/github"
	log "github.com/sirupsen/logrus"
)

type WriteCounter struct {
	Written  int
	Total    int
	Label    string
	nWrites  int
	progress bool
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Written += n
	wc.nWrites++
	if wc.progress && wc.nWrites%300 == 0 {
		log.Infof("downloading quarto version %s ... (%s/%s)", wc.Label,
			humanize.Bytes(uint64(wc.Written)),
			humanize.Bytes(uint64(wc.Total)))
	}
	if wc.Written == wc.Total {
		log.Infof("completed downloading quarto version %s", wc.Label)
	}
	return n, nil
}

// DownloadReleaseAsset downloads the release asset for an edition+platform to a
// temp file and returns the path to the written file. When the edition requires
// it, the download is verified against its ".sha256" asset before returning.
func DownloadReleaseAsset(client *github.Client, e edition.Edition, tag string, p edition.Platform, progress bool) (string, error) {
	suffix, err := e.AssetSuffix(p)
	if err != nil {
		return "", err
	}
	release, err := GetRelease(client, e.Owner, e.Repo, tag)
	if err != nil {
		return "", fmt.Errorf("failed to get release %s: %w", tag, err)
	}
	asset := findAssetBySuffix(release.Assets, suffix)
	if asset == nil {
		return "", fmt.Errorf("no release asset found for %s/%s with suffix '%s'. This version may not support your platform/architecture. Available assets: %s",
			p.OS, p.Arch, suffix, listAvailableAssets(release.Assets))
	}
	log.Tracef("fetching information to download release asset from %s\n", asset.GetBrowserDownloadURL())
	start := time.Now()
	rc, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), e.Owner, e.Repo, asset.GetID(), http.DefaultClient)
	log.Tracef("done fetching release asset information in %s\n", time.Since(start))
	if err != nil {
		return "", err
	}
	defer rc.Close()
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("*-%s", asset.GetName()))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	log.Tracef("starting to copy release asset to %s\n", tmpFile.Name())
	start = time.Now()
	counter := &WriteCounter{Total: asset.GetSize(), Label: tag, progress: progress}
	wb, err := io.Copy(tmpFile, io.TeeReader(rc, counter))
	log.Tracef("done copying release asset in %s\n", time.Since(start))
	if err != nil {
		return "", err
	}
	log.Debugf("downloaded %d bytes from release asset at url %s\n", wb, asset.GetBrowserDownloadURL())

	if e.VerifyChecksum {
		if err := verifyAssetChecksum(client, e, release.Assets, asset.GetName(), tmpFile.Name()); err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}
		log.Infof("verified sha256 checksum for %s", asset.GetName())
	}
	return tmpFile.Name(), nil
}

func findAssetBySuffix(assets []*github.ReleaseAsset, suffix string) *github.ReleaseAsset {
	if suffix == "" {
		return nil
	}
	for _, asset := range assets {
		if strings.HasSuffix(asset.GetName(), suffix) {
			return asset
		}
	}
	return nil
}

func listAvailableAssets(assets []*github.ReleaseAsset) string {
	var names []string
	for _, asset := range assets {
		name := asset.GetName()
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// verifyAssetChecksum downloads the "<assetName>.sha256" sibling asset and
// verifies the downloaded file against it.
func verifyAssetChecksum(client *github.Client, e edition.Edition, assets []*github.ReleaseAsset, assetName, filePath string) error {
	checksumName := assetName + ".sha256"
	var checksumAsset *github.ReleaseAsset
	for _, a := range assets {
		if a.GetName() == checksumName {
			checksumAsset = a
			break
		}
	}
	if checksumAsset == nil {
		return fmt.Errorf("checksum asset %s not found; refusing to install unverified download", checksumName)
	}
	rc, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), e.Owner, e.Repo, checksumAsset.GetID(), http.DefaultClient)
	if err != nil {
		return fmt.Errorf("downloading checksum asset %s: %w", checksumName, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("reading checksum asset %s: %w", checksumName, err)
	}
	return verifyFileChecksum(filePath, string(data))
}

// verifyFileChecksum compares the SHA256 of filePath against the first
// whitespace-separated field of a standard "<hexdigest>  <name>" checksum line.
func verifyFileChecksum(filePath, checksumContent string) error {
	fields := strings.Fields(checksumContent)
	if len(fields) == 0 {
		return fmt.Errorf("malformed checksum content for %s", filePath)
	}
	want := fields[0]
	got, err := sha256File(filePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filePath, want, got)
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

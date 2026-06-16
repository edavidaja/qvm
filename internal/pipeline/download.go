package pipeline

import (
	"os"
	"path/filepath"

	"github.com/dpastoor/qvm/internal/config"
	"github.com/dpastoor/qvm/internal/edition"
	"github.com/dpastoor/qvm/internal/gh"
	"github.com/dpastoor/qvm/internal/unarchive"
)

// DownloadReleaseVersion downloads and extracts the appropriate asset for an
// edition and platform into the edition's versions dir, returning the install
// directory.
func DownloadReleaseVersion(e edition.Edition, release string, p edition.Platform, progress bool) (string, error) {
	client := gh.NewClient(os.Getenv("GITHUB_PAT"))
	dlPath, err := gh.DownloadReleaseAsset(client, e, release, p, progress)
	if err != nil {
		return "", err
	}
	dirPath := filepath.Join(config.GetPathToVersionsDir(e), release)
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return "", err
	}
	dlFile, err := os.Open(dlPath)
	if err != nil {
		return "", err
	}
	err = unarchive.Unarchive(dlFile, dirPath)
	if err != nil {
		return "", err
	}
	return dirPath, err
}

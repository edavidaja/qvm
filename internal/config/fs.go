package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	log "github.com/sirupsen/logrus"

	"github.com/dpastoor/qvm/internal/edition"
)

func GetConfigPath() (string, error) {
	return xdg.ConfigFile("qvm/config.yml")
}

func GetRootConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "qvm")
}
func GetRootDataPath() string {
	return filepath.Join(xdg.DataHome, "qvm")
}

// GetPathToActiveBinDir returns the path to the active version's bin directory
// if that directory does not exist, it will be created with perms 700
func GetPathToActiveBinDir() string {
	path := filepath.Join(xdg.ConfigHome, "qvm", "bin")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			fmt.Println("unable to create active bin dir, there could be issues when attempting to access!")
		}
	}
	return path
}

// GetPathToActiveExe returns the path to the active symlink for an edition
// inside the shared active bin dir (e.g. .../qvm/bin/quarto or .../qvm/bin/q2).
func GetPathToActiveExe(e edition.Edition) string {
	return filepath.Join(GetPathToActiveBinDir(), e.ExeName())
}
func GetPathToVersionsDir(e edition.Edition) string {
	// if running as root, install q1 to /opt/quarto as an admin helper
	if runtime.GOOS == "linux" && os.Getuid() == 0 && e.Kind == edition.Q1 {
		return filepath.Join("/opt", "quarto")
	}
	return filepath.Join(xdg.DataHome, "qvm", e.VersionsDirName)
}

func GetActiveVersion(e edition.Edition) (string, error) {
	path, err := filepath.EvalSymlinks(GetPathToActiveExe(e))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("no active quarto version detected")
		}
		return "", err
	}
	// path resolves to <versionsDir>/<version>/<ExeRelPath>; strip the
	// ExeRelPath components to land on the version dir, then take its base.
	versionDir := path
	for range strings.Split(filepath.ToSlash(e.ExeRelPath()), "/") {
		versionDir = filepath.Dir(versionDir)
	}
	version := filepath.Base(versionDir)
	if version == "." || version == string(filepath.Separator) {
		return "", errors.New(`
		something went wrong with the active version detection,
		please contact the developers`)
	}
	return version, nil
}

// GetInstalledVersions returns a map of installed versions for an edition where
// the key is the version and the value is the path to its executable.
func GetInstalledVersions(e edition.Edition) (map[string]string, error) {
	iv := make(map[string]string)
	entries, err := os.ReadDir(GetPathToVersionsDir(e))
	if err != nil {
		return iv, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			exePath := filepath.Join(GetPathToVersionsDir(e), entry.Name(), e.ExeRelPath())
			if _, err := os.Stat(exePath); err == nil {
				iv[entry.Name()] = exePath
			} else {
				log.Warn("could not find expected executable for version: ", entry.Name())
			}
		}
	}
	return iv, nil
}

func Read() (Config, string, error) {
	// for now don't check on error until consider what type of
	// logging - this should be completely optional anyway
	globalConfigPath, err := xdg.SearchConfigFile("qvm/config.yml")
	if err != nil {
		// this message is and print out all the paths that were searched
		// for example, by mangling the name to `onfig.yml` to see what the
		// error was, got the following result:
		//
		// could not locate `config.yml` in any of the following paths: /Users/devinp/Library/Application Support/qvm, /Users/devinp/Library/Preferences/qvm, /Library/Application Support/qvm, /Library/Preferences/qvm
		// exit status 1
		return Config{}, "", err
	}
	cfg, err := read(globalConfigPath)
	return cfg, globalConfigPath, err
}

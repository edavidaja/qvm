package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/dpastoor/qvm/internal/edition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempXDG points xdg.DataHome at a temp dir for the duration of a test.
func withTempXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := xdg.DataHome
	xdg.DataHome = dir
	t.Cleanup(func() { xdg.DataHome = orig })
	return dir
}

func TestGetPathToVersionsDir(t *testing.T) {
	data := withTempXDG(t)
	assert.Equal(t, filepath.Join(data, "qvm", "versions"),
		GetPathToVersionsDir(edition.For(edition.Q1)))
	assert.Equal(t, filepath.Join(data, "qvm", "versions-q2"),
		GetPathToVersionsDir(edition.For(edition.Q2)))
}

func TestGetInstalledVersions_PerEdition(t *testing.T) {
	withTempXDG(t)
	q1 := edition.For(edition.Q1)
	q2 := edition.For(edition.Q2)

	// q1 layout: versions/v1.8.25/bin/quarto
	q1exe := filepath.Join(GetPathToVersionsDir(q1), "v1.8.25", q1.ExeRelPath())
	require.NoError(t, os.MkdirAll(filepath.Dir(q1exe), 0o755))
	require.NoError(t, os.WriteFile(q1exe, []byte("x"), 0o755))

	// q2 layout: versions-q2/v0.1.1/q2
	q2exe := filepath.Join(GetPathToVersionsDir(q2), "v0.1.1", q2.ExeRelPath())
	require.NoError(t, os.MkdirAll(filepath.Dir(q2exe), 0o755))
	require.NoError(t, os.WriteFile(q2exe, []byte("x"), 0o755))

	ivQ1, err := GetInstalledVersions(q1)
	require.NoError(t, err)
	assert.Contains(t, ivQ1, "v1.8.25")
	assert.NotContains(t, ivQ1, "v0.1.1")

	ivQ2, err := GetInstalledVersions(q2)
	require.NoError(t, err)
	assert.Contains(t, ivQ2, "v0.1.1")
	assert.NotContains(t, ivQ2, "v1.8.25")
}

package gh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v73/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAssetBySuffix(t *testing.T) {
	assets := []*github.ReleaseAsset{
		{Name: github.String("quarto-1.4.553-linux-amd64.tar.gz")},
		{Name: github.String("quarto-1.4.553-linux-arm64.tar.gz")},
		{Name: github.String("quarto-1.4.553-macos.tar.gz")},
		{Name: github.String("q2-0.1.1-darwin_arm64.tar.gz")},
	}
	tests := []struct {
		name      string
		suffix    string
		wantFound bool
		wantName  string
	}{
		{"q1 linux amd64", "linux-amd64.tar.gz", true, "quarto-1.4.553-linux-amd64.tar.gz"},
		{"q1 macos", "macos.tar.gz", true, "quarto-1.4.553-macos.tar.gz"},
		{"q2 darwin arm64", "darwin_arm64.tar.gz", true, "q2-0.1.1-darwin_arm64.tar.gz"},
		{"missing", "win.zip", false, ""},
		{"empty suffix never matches", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := findAssetBySuffix(assets, tt.suffix)
			if tt.wantFound {
				require.NotNil(t, asset)
				assert.Equal(t, tt.wantName, asset.GetName())
			} else {
				assert.Nil(t, asset)
			}
		})
	}
}

func TestVerifyFileChecksum(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.tar.gz")
	require.NoError(t, os.WriteFile(file, []byte("hello qvm\n"), 0o644))

	t.Run("matches when correct", func(t *testing.T) {
		good, err := sha256File(file)
		require.NoError(t, err)
		content := good + "  payload.tar.gz\n"
		assert.NoError(t, verifyFileChecksum(file, content))
	})
	t.Run("errors on mismatch", func(t *testing.T) {
		content := "0000000000000000000000000000000000000000000000000000000000000000  payload.tar.gz\n"
		err := verifyFileChecksum(file, content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})
	t.Run("errors on malformed content", func(t *testing.T) {
		assert.Error(t, verifyFileChecksum(file, "   \n"))
	})
}

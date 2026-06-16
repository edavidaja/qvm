package edition

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromFlag(t *testing.T) {
	assert.Equal(t, Q1, FromFlag(false).Kind)
	assert.Equal(t, Q2, FromFlag(true).Kind)
}

func TestForIdentity(t *testing.T) {
	q1 := For(Q1)
	assert.Equal(t, "quarto-dev", q1.Owner)
	assert.Equal(t, "quarto-cli", q1.Repo)
	assert.Equal(t, "versions", q1.VersionsDirName)
	assert.False(t, q1.VerifyChecksum)

	q2 := For(Q2)
	assert.Equal(t, "quarto-dev", q2.Owner)
	assert.Equal(t, "q2", q2.Repo)
	assert.Equal(t, "versions-q2", q2.VersionsDirName)
	assert.True(t, q2.VerifyChecksum)
}

func TestAssetSuffix(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		platform Platform
		want     string
		wantErr  bool
	}{
		{"q1 darwin universal", Q1, Platform{"darwin", "arm64"}, "macos.tar.gz", false},
		{"q1 linux amd64", Q1, Platform{"linux", "amd64"}, "linux-amd64.tar.gz", false},
		{"q1 linux arm64", Q1, Platform{"linux", "arm64"}, "linux-arm64.tar.gz", false},
		{"q1 windows", Q1, Platform{"windows", "amd64"}, "win.zip", false},
		{"q1 rhel7", Q1, Platform{"rhel7", "amd64"}, "linux-rhel7-amd64.tar.gz", false},
		{"q1 unsupported os", Q1, Platform{"freebsd", "amd64"}, "", true},
		{"q2 darwin arm64", Q2, Platform{"darwin", "arm64"}, "darwin_arm64.tar.gz", false},
		{"q2 darwin amd64", Q2, Platform{"darwin", "amd64"}, "darwin_amd64.tar.gz", false},
		{"q2 linux amd64", Q2, Platform{"linux", "amd64"}, "linux_amd64.tar.gz", false},
		{"q2 linux arm64", Q2, Platform{"linux", "arm64"}, "linux_arm64.tar.gz", false},
		{"q2 windows amd64", Q2, Platform{"windows", "amd64"}, "windows_amd64.zip", false},
		{"q2 windows arm64 unsupported", Q2, Platform{"windows", "arm64"}, "", true},
		{"q2 rhel7 unsupported", Q2, Platform{"rhel7", "amd64"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := For(tt.kind).AssetSuffix(tt.platform)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_runAtGitRef(t *testing.T) {
	dir := t.TempDir()
	fooPath := filepath.Join(dir, "foo")
	err := os.WriteFile(fooPath, []byte("OG content"), 0o600)
	require.NoError(t, err)
	mustGit(t, dir, "init")
	mustGit(t, dir, "add", "foo")
	mustGit(t, dir, "commit", "-m", "ignore me")
	untrackedPath := filepath.Join(dir, "untracked")
	err = os.WriteFile(untrackedPath, []byte("untracked"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(fooPath, []byte("new content"), 0o600)
	require.NoError(t, err)
	fn := func(workDir string) {
		var got []byte
		_, err = os.ReadFile(filepath.Join(workDir, "untracked"))
		require.Error(t, err)
		wdFooPath := filepath.Join(workDir, "foo")
		got, err = os.ReadFile(wdFooPath)
		require.NoError(t, err)
		require.Equal(t, "OG content", string(got))
	}
	err = runAtGitRef(nil, "git", dir, "HEAD", fn)
	require.NoError(t, err)
	got, err := os.ReadFile(fooPath)
	require.NoError(t, err)
	require.Equal(t, "new content", string(got))
}

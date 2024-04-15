package internal

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustSetEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		assert.NoError(t, os.Setenv(k, v))
	}
}

func mustGit(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	mustSetEnv(t, map[string]string{
		"GIT_AUTHOR_NAME":     "author",
		"GIT_AUTHOR_EMAIL":    "author@localhost",
		"GIT_COMMITTER_NAME":  "committer",
		"GIT_COMMITTER_EMAIL": "committer@localhost",
	})
	got, err := runGitCmd(nil, "git", repoPath, args...)
	assert.NoErrorf(t, err, "error running git:\noutput: %v", string(got))
	return got
}

func mustGo(t *testing.T, path string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = path
	got, err := cmd.Output()
	assert.NoErrorf(t, err, "error running go:\noutput: %v", string(got))
	return got
}

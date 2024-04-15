package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/benchdiff/pkg/benchstatter"
)

func setupTestRepo(t *testing.T, path string) {
	t.Helper()
	mustGo(t, path, "mod", "init", "bindiff.test")
	ex1 := filepath.Join(path, "ex1.go")
	ex1test := filepath.Join(path, "ex1_test.go")
	err := os.WriteFile(ex1, []byte(ex1Rev1), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(ex1test, []byte(ex1Bench), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(path, ".gitignore"), []byte("tmp/\n"), 0o600)
	require.NoError(t, err)
	mustGit(t, path, "init")
	err = os.MkdirAll(filepath.Join(path, "tmp"), 0o700)
	require.NoError(t, err)
	mustGit(t, path, "add", ".")
	mustGit(t, path, "commit", "-m", "initial commit")
	err = os.WriteFile(ex1, []byte(ex1Rev2), 0o600)
	require.NoError(t, err)
}

func testInDir(t *testing.T, dir string) {
	t.Helper()
	pwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		t.Helper()
		require.NoError(t, os.Chdir(pwd))
	})
}

func TestBenchdiff_Run(t *testing.T) {
	dir := t.TempDir()
	setupTestRepo(t, dir)
	testInDir(t, dir)
	differ := Benchdiff{
		GitCmd:     "git",
		BenchCmd:   "go",
		BenchArgs:  "test -bench . -benchmem -count 10 -benchtime 10x .",
		ResultsDir: "./tmp",
		BaseRef:    "HEAD",
		Path:       ".",
		Benchstat:  &benchstatter.Benchstat{},
	}
	_, err := differ.Run()
	require.NoError(t, err)
}

var ex1Rev1 = `
package ex1

import (
	"time"
)

var globalBytes []byte

func doNothing() {
	time.Sleep(10 * time.Millisecond)
	newBytes := []byte("0")
	globalBytes = append(globalBytes, newBytes...)
}
`

var ex1Rev2 = `
package ex1

import (
	"time"
)

var globalBytes []byte

func doNothing() {
	time.Sleep(1 * time.Millisecond)
	newBytes := []byte("1123456789")
	globalBytes = append(globalBytes, newBytes...)
}
`

var ex1Bench = `
package ex1

import (
	"testing"
)

func BenchmarkDoNothing(b *testing.B) {
	globalBytes = []byte{}
	for i := 0; i < b.N; i++ {
		doNothing()
	}
}
`

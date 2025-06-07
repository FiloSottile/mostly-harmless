package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutable(t *testing.T) {
	dirs := []string{
		"/usr/local/bin",
		"/etc/monitor.d",
		"/etc/local.d",
		"/etc/init.d",
		"/etc/profile.d",
	}

	ff, _ := os.ReadDir(filepath.Join("..", "root", "/etc/periodic"))
	for _, f := range ff {
		if f.IsDir() {
			dirs = append(dirs, filepath.Join("/etc/periodic", f.Name()))
		}
	}

	var gotFiles bool
	// Check that all files in those directories are executable.
	for _, dir := range dirs {
		files, err := os.ReadDir(filepath.Join("..", "root", dir))
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, err := file.Info()
			if err != nil {
				t.Fatalf("Failed to get file info: %v", err)
			}
			if info.Mode()&0111 == 0 {
				t.Errorf("File %q is not executable", filepath.Join(dir, file.Name()))
			}
			gotFiles = true
		}
	}
	if !gotFiles {
		t.Error("No files found in directories")
	}
}

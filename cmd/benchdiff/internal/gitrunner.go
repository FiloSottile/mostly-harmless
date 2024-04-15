package internal

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func runGitCmd(debug *log.Logger, gitCmd, repoPath string, args ...string) ([]byte, error) {
	var stdout bytes.Buffer
	cmd := exec.Command(gitCmd, args...)
	cmd.Stdout = &stdout
	cmd.Dir = repoPath
	err := runCmd(cmd, debug)
	return bytes.TrimSpace(stdout.Bytes()), err
}

func runAtGitRef(debug *log.Logger, gitCmd, repoPath, ref string, fn func(path string)) error {
	worktree, err := os.MkdirTemp("", "benchdiff")
	if err != nil {
		return err
	}
	defer func() {
		rErr := os.RemoveAll(worktree)
		if rErr != nil {
			fmt.Printf("Could not delete temp directory: %s\n", worktree)
		}
	}()

	_, err = runGitCmd(debug, gitCmd, repoPath, "worktree", "add", "--quiet", "--detach", worktree, ref)
	if err != nil {
		return err
	}

	defer func() {
		_, cerr := runGitCmd(debug, gitCmd, repoPath, "worktree", "remove", worktree)
		if cerr != nil {
			if exitErr, ok := cerr.(*exec.ExitError); ok {
				fmt.Println(string(exitErr.Stderr))
			}
			fmt.Println(cerr)
		}
	}()
	fn(worktree)
	return nil
}

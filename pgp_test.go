package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func init() {
	stdinPwd = true
}

func tFatalIfErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func getBackup(t *testing.T) string {
	f, err := os.Open("testdata/pgp.backup.txt")
	tFatalIfErr(err, t)
	bak, err := ioutil.ReadAll(f)
	tFatalIfErr(err, t)
	return string(bak)
}

func TestPGPBackupArmor(t *testing.T) {
	if os.Getenv("RUN") == "1" {
		os.Args = []string{os.Args[0], "testdata/pgp.secret.password.armor.asc"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPGPBackupArmor$")
	cmd.Env = append(os.Environ(), "RUN=1")
	if testing.Verbose() {
		cmd.Stderr = os.Stderr
	}
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	tFatalIfErr(cmd.Start(), t)
	stdin.Write([]byte("password\n"))
	r, _ := ioutil.ReadAll(stdout)
	res := string(r)[:len(r)-len("PASS\n")]
	if getBackup(t) != res {
		t.Log("\n", res)
		t.Fail()
	}
	tFatalIfErr(cmd.Wait(), t)
}

func TestPGPBackup(t *testing.T) {
	if os.Getenv("RUN") == "1" {
		os.Args = []string{os.Args[0], "testdata/pgp.secret.password.asc"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPGPBackup$")
	cmd.Env = append(os.Environ(), "RUN=1")
	if testing.Verbose() {
		cmd.Stderr = os.Stderr
	}
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	tFatalIfErr(cmd.Start(), t)
	stdin.Write([]byte("password\n"))
	r, _ := ioutil.ReadAll(stdout)
	res := string(r)[:len(r)-len("PASS\n")]
	if getBackup(t) != res {
		t.Log("\n", res)
		t.Fail()
	}
	tFatalIfErr(cmd.Wait(), t)
}

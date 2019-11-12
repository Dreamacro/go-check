package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

func execSync(pwd string, command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = pwd

	buf := &bytes.Buffer{}
	bufErr := &bytes.Buffer{}
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go io.Copy(buf, stdout)
	go io.Copy(bufErr, stderr)
	if err := cmd.Run(); err != nil {
		return nil, errors.New(string(bufErr.Bytes()))
	}
	return buf.Bytes(), nil
}

func Exec(pwd string) ([]byte, error) {
	return execSync(pwd, "go", "list", "-u", "-m", "-json", "all")
}

func Upgrade(pwd string, pkgs []*Package) ([]byte, error) {
	args := []string{"get"}
	for _, pkg := range pkgs {
		args = append(args, fmt.Sprintf("%s@%s", pkg.Path, pkg.Update.Version))
	}

	return execSync(pwd, "go", args...)
}

func Tidy(pwd string) ([]byte, error) {
	return execSync(pwd, "go", "mod", "tidy")
}

package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"golang.org/x/tools/go/packages"
)

// Module provides module information for a package.
type Module struct {
	Path      string                // module path
	Version   string                // module version
	Replace   *Module               // replaced by this module
	Time      *time.Time            // time version was created
	Update    *Module               // available update, if any (with -u)
	Main      bool                  // is this the main module?
	Indirect  bool                  // is this module only an indirect dependency of main module?
	Dir       string                // directory holding files for this module, if any
	GoMod     string                // path to go.mod file used when loading this module, if any
	GoVersion string                // go version used in module
	Error     *packages.ModuleError // error loading module
}

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
		return nil, errors.New(bufErr.String())
	}
	return buf.Bytes(), nil
}

func GetModuleUpdate(pwd, module string) (*Module, error) {
	buf, err := execSync(pwd, "go", "list", "-u", "-m", "-mod=mod", "-json", module)
	if err != nil {
		return nil, err
	}

	pkg := Module{}
	if err := json.Unmarshal(buf, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

func Upgrade(pwd string, pkgs []*Module) ([]byte, error) {
	args := []string{"get"}
	for _, pkg := range pkgs {
		args = append(args, fmt.Sprintf("%s@%s", pkg.Path, pkg.Update.Version))
	}

	return execSync(pwd, "go", args...)
}

func Tidy(pwd string) ([]byte, error) {
	return execSync(pwd, "go", "mod", "tidy")
}

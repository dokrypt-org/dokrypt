package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func runDokrypt(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := projectRoot(t)
	cmd := exec.Command("go", append([]string{"run", "./cmd/dokrypt"}, args...)...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLI_Version(t *testing.T) {
	out, err := runDokrypt(t, "version")
	if err != nil {
		t.Fatalf("dokrypt version failed: %v\nOutput: %s", err, out)
	}
	if !strings.Contains(strings.ToLower(out), "dokrypt") {
		t.Errorf("version output missing 'dokrypt': %s", out)
	}
}

func TestCLI_Help(t *testing.T) {
	out, err := runDokrypt(t, "--help")
	if err != nil {
		t.Fatalf("dokrypt --help failed: %v\nOutput: %s", err, out)
	}
	for _, subcmd := range []string{"init", "up", "down", "status"} {
		if !strings.Contains(out, subcmd) {
			t.Errorf("help output missing command %q", subcmd)
		}
	}
}

func TestCLI_Init(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")
	out, err := runDokrypt(t, "init", projectDir, "--template", "evm-basic", "--no-git")
	if err != nil {
		t.Fatalf("dokrypt init failed: %v\nOutput: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "dokrypt.yaml")); os.IsNotExist(err) {
		t.Error("dokrypt.yaml not created")
	}
}

func TestCLI_ConfigValidate(t *testing.T) {
	root := projectRoot(t)
	cfgPath := filepath.Join(root, "internal", "template", "builtin", "templates", "evm-basic", "dokrypt.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Skip("template config not found")
	}
	out, err := runDokrypt(t, "config", "validate", "--config", cfgPath)
	if err != nil {
		t.Fatalf("config validate failed: %v\nOutput: %s", err, out)
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	_, err := runDokrypt(t, "nonexistent-command")
	if err == nil {
		t.Error("dokrypt should exit with error for unknown command")
	}
}

func TestCLI_StatusWithoutEnvironment(t *testing.T) {
	out, err := runDokrypt(t, "status")
	_ = err
	_ = out
}

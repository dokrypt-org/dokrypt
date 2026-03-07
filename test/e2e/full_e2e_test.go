package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var dokryptBinary string

var testProjectDir string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "dokrypt-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	binName := "dokrypt"
	if runtime.GOOS == "windows" {
		binName = "dokrypt.exe"
	}
	dokryptBinary = filepath.Join(tmp, binName)

	root := findProjectRoot()
	if root == "" {
		fmt.Fprintln(os.Stderr, "cannot find project root (go.mod)")
		os.Exit(1)
	}

	fmt.Println("Building dokrypt binary...")
	cmd := exec.Command("go", "build", "-o", dokryptBinary, "./cmd/dokrypt")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Binary built at: %s\n\n", dokryptBinary)

	parentDir, _ := os.MkdirTemp("", "dokrypt-e2e-*")
	testProjectDir = filepath.Join(parentDir, "e2etest")

	code := m.Run()

	cleanup := exec.Command(dokryptBinary, "down", "--volumes")
	cleanup.Dir = testProjectDir
	cleanup.Run()

	os.RemoveAll(tmp)
	os.RemoveAll(testProjectDir)
	os.Exit(code)
}

func findProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func run(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(dokryptBinary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runOK(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := run(t, dir, args...)
	if err != nil {
		t.Fatalf("dokrypt %s failed: %v\nOutput:\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func runFail(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := run(t, dir, args...)
	if err == nil {
		t.Fatalf("dokrypt %s should have failed but succeeded\nOutput:\n%s", strings.Join(args, " "), out)
	}
	return out
}

func assertContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Errorf("expected output to contain %q, got:\n%s", expected, output)
	}
}


func dockerAvailable(t *testing.T) {
	t.Helper()
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker not available, skipping E2E tests")
	}
}

func TestE2E_Phase1_Standalone(t *testing.T) {
	t.Run("Version", func(t *testing.T) {
		out := runOK(t, ".", "version")
		assertContains(t, out, "dokrypt")
		assertContains(t, out, "commit:")
		assertContains(t, out, "built:")
		assertContains(t, out, "go:")
		assertContains(t, out, "os/arch:")
	})

	t.Run("Help", func(t *testing.T) {
		out := runOK(t, ".", "--help")
		for _, cmd := range []string{"init", "up", "down", "status", "chain", "snapshot", "bridge", "fork", "accounts", "template", "config", "doctor", "marketplace", "plugin", "test", "version", "exec", "logs", "restart"} {
			assertContains(t, out, cmd)
		}
	})

	t.Run("UnknownCommand", func(t *testing.T) {
		_, err := run(t, ".", "nonexistent-command-xyz")
		if err == nil {
			t.Error("unknown command should fail")
		}
	})

	t.Run("Doctor", func(t *testing.T) {
		out := runOK(t, ".", "doctor")
		assertContains(t, out, "Dokrypt Doctor")
		assertContains(t, out, "Platform")
		assertContains(t, out, "Docker installed")
		assertContains(t, out, "Disk space")
	})

	t.Run("TemplateList", func(t *testing.T) {
		out := runOK(t, ".", "template", "list")
		assertContains(t, out, "evm-basic")
		assertContains(t, out, "evm-defi")
		assertContains(t, out, "evm-nft")
		assertContains(t, out, "evm-dao")
		assertContains(t, out, "evm-token")
	})

	t.Run("TemplateInfoEvmBasic", func(t *testing.T) {
		out := runOK(t, ".", "template", "info", "evm-basic")
		assertContains(t, out, "evm-basic")
	})

	t.Run("TemplateInfoNotFound", func(t *testing.T) {
		runFail(t, ".", "template", "info", "nonexistent-template-xyz")
	})

	t.Run("PluginList_Empty", func(t *testing.T) {
		out := runOK(t, ".", "plugin", "list")
		_ = out
	})

	t.Run("MarketplaceSearch", func(t *testing.T) {
		_, _ = run(t, ".", "marketplace", "search", "test")
	})

	t.Run("MarketplaceBrowse", func(t *testing.T) {
		_, _ = run(t, ".", "marketplace", "browse")
	})

	t.Run("ConfigInit", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := runOK(t, tmpDir, "config", "init")
		assertContains(t, out, "dokrypt.yaml")

		if _, err := os.Stat(filepath.Join(tmpDir, "dokrypt.yaml")); os.IsNotExist(err) {
			t.Error("dokrypt.yaml not created by config init")
		}
	})
}

func TestE2E_Phase2_Init(t *testing.T) {
	t.Run("InitEvmBasic", func(t *testing.T) {
		out := runOK(t, filepath.Dir(testProjectDir), "init", testProjectDir, "--template", "evm-basic", "--no-git")
		assertContains(t, out, "Scaffolding")

		mustExist(t, filepath.Join(testProjectDir, "dokrypt.yaml"))
		mustExist(t, filepath.Join(testProjectDir, "foundry.toml"))
		mustExist(t, filepath.Join(testProjectDir, "README.md"))
	})

	t.Run("InitDuplicateProject", func(t *testing.T) {
		runFail(t, filepath.Dir(testProjectDir), "init", testProjectDir, "--template", "evm-basic", "--no-git")
	})

	t.Run("ConfigValidate", func(t *testing.T) {
		out := runOK(t, testProjectDir, "config", "validate")
		assertContains(t, out, "valid")
	})

	t.Run("ConfigShow", func(t *testing.T) {
		out := runOK(t, testProjectDir, "config", "show")
		assertContains(t, out, "ethereum")
		assertContains(t, out, "anvil")
	})

	t.Run("InitEvmBasicInTempDir", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectPath := filepath.Join(tmpDir, "test-basic")
		out := runOK(t, tmpDir, "init", projectPath, "--template", "evm-basic", "--no-git")
		assertContains(t, out, "Scaffolding")
		mustExist(t, filepath.Join(projectPath, "dokrypt.yaml"))
	})

	t.Run("InitEvmDefi", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectPath := filepath.Join(tmpDir, "test-defi")
		out := runOK(t, tmpDir, "init", projectPath, "--template", "evm-defi", "--no-git")
		assertContains(t, out, "Scaffolding")
		mustExist(t, filepath.Join(projectPath, "dokrypt.yaml"))
	})
}

func TestE2E_Phase3_StackLifecycle(t *testing.T) {
	dockerAvailable(t)

	if _, err := os.Stat(filepath.Join(testProjectDir, "dokrypt.yaml")); os.IsNotExist(err) {
		runOK(t, filepath.Dir(testProjectDir), "init", testProjectDir, "--template", "evm-basic", "--no-git")
	}

	t.Run("Up", func(t *testing.T) {
		out := runOK(t, testProjectDir, "up", "--detach", "--timeout", "3m")
		assertContains(t, out, "Ready")
		assertContains(t, out, "running")
	})

	time.Sleep(3 * time.Second)

	t.Run("Status", func(t *testing.T) {
		out := runOK(t, testProjectDir, "status")
		assertContains(t, out, "ethereum")
		assertContains(t, out, "Ready")
	})

	t.Run("StatusJSON", func(t *testing.T) {
		out := runOK(t, testProjectDir, "status", "--json")
		_ = out
	})

	t.Run("StatusServiceFilter", func(t *testing.T) {
		out := runOK(t, testProjectDir, "status", "--service", "ethereum")
		assertContains(t, out, "ethereum")
	})

	t.Run("StatusServiceNotFound", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "status", "--service", "nonexistent")
		assertContains(t, out, "not found")
	})

	t.Run("ChainInfo", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "info")
		assertContains(t, out, "Chain ID")
		assertContains(t, out, "RPC URL")
		assertContains(t, out, "Block")
	})

	t.Run("ChainMine_Default", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "mine")
		assertContains(t, out, "Mined")
		assertContains(t, out, "block")
	})

	t.Run("ChainMine_Multiple", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "mine", "5")
		assertContains(t, out, "Mined 5")
	})

	t.Run("ChainSetBalance", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "set-balance", "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "99999")
		assertContains(t, out, "Balance")
		assertContains(t, out, "99999")
	})

	t.Run("ChainTimeTravel", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "time-travel", "3600")
		assertContains(t, out, "Advanced")
	})

	t.Run("ChainSetGasPrice", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "set-gas-price", "20")
		assertContains(t, out, "Gas price")
		assertContains(t, out, "20")
	})

	t.Run("ChainImpersonate", func(t *testing.T) {
		vitalik := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
		out := runOK(t, testProjectDir, "chain", "impersonate", vitalik)
		assertContains(t, out, "impersonating")
	})

	t.Run("ChainStopImpersonating", func(t *testing.T) {
		vitalik := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
		out := runOK(t, testProjectDir, "chain", "stop-impersonating", vitalik)
		assertContains(t, out, "Stopped impersonating")
	})

	t.Run("ChainReset", func(t *testing.T) {
		out, err := run(t, testProjectDir, "chain", "reset")
		if err != nil {
			assertContains(t, out, "Forking not enabled")
		} else {
			assertContains(t, out, "reset")
		}
	})

	runOK(t, testProjectDir, "chain", "mine", "3")

	t.Run("AccountsList", func(t *testing.T) {
		out := runOK(t, testProjectDir, "accounts", "list")
		assertContains(t, out, "0x")
		assertContains(t, out, "ETH")
	})

	t.Run("AccountsFund", func(t *testing.T) {
		out := runOK(t, testProjectDir, "accounts", "fund", "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "500")
		assertContains(t, out, "Funded")
		assertContains(t, out, "500")
	})

	t.Run("AccountsGenerate", func(t *testing.T) {
		out := runOK(t, testProjectDir, "accounts", "generate", "3")
		assertContains(t, out, "Generated")
	})

	t.Run("ExecCommand", func(t *testing.T) {
		out, err := run(t, testProjectDir, "exec", "ethereum", "echo", "hello-from-container")
		if err != nil {
			t.Logf("exec returned error (may be expected): %v\nOutput: %s", err, out)
		} else {
			assertContains(t, out, "hello-from-container")
		}
	})

	t.Run("ExecServiceNotFound", func(t *testing.T) {
		out := runFail(t, testProjectDir, "exec", "nonexistent-service", "echo", "hi")
		assertContains(t, out, "not found")
	})

	t.Run("Logs", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "logs", "--tail", "5")
		_ = out
	})

	t.Run("LogsService", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "logs", "--service", "ethereum", "--tail", "5")
		_ = out
	})

	t.Run("SnapshotSave", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "save", "e2e-snap-1", "--description", "E2E test snapshot")
		assertContains(t, out, "e2e-snap-1")
		assertContains(t, out, "saved")
	})

	t.Run("SnapshotList", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "list")
		assertContains(t, out, "e2e-snap-1")
	})

	runOK(t, testProjectDir, "chain", "mine", "10")

	t.Run("SnapshotSave_Second", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "save", "e2e-snap-2", "--description", "Second snapshot")
		assertContains(t, out, "e2e-snap-2")
	})

	t.Run("SnapshotDiff", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "diff", "e2e-snap-1", "e2e-snap-2")
		_ = out
	})

	t.Run("SnapshotRestore", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "restore", "e2e-snap-1")
		assertContains(t, out, "Restored")
	})

	t.Run("SnapshotExport", func(t *testing.T) {
		exportPath := filepath.Join(t.TempDir(), "e2e-export.tar.gz")
		out := runOK(t, testProjectDir, "snapshot", "export", "e2e-snap-1", exportPath)
		assertContains(t, out, "exported")

		info, err := os.Stat(exportPath)
		if err != nil {
			t.Fatalf("export file not created: %v", err)
		}
		if info.Size() == 0 {
			t.Error("export file is empty")
		}
	})

	t.Run("SnapshotImport", func(t *testing.T) {
		exportPath := filepath.Join(t.TempDir(), "e2e-import.tar.gz")
		runOK(t, testProjectDir, "snapshot", "export", "e2e-snap-2", exportPath)

		runOK(t, testProjectDir, "snapshot", "delete", "e2e-snap-2")

		out := runOK(t, testProjectDir, "snapshot", "import", exportPath)
		assertContains(t, out, "imported")
	})

	t.Run("SnapshotDelete", func(t *testing.T) {
		out := runOK(t, testProjectDir, "snapshot", "delete", "e2e-snap-1")
		assertContains(t, out, "deleted")
	})

	t.Run("SnapshotRestoreNotFound", func(t *testing.T) {
		runFail(t, testProjectDir, "snapshot", "restore", "nonexistent-snapshot")
	})

	t.Run("BridgeStatus", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "bridge", "status")
		_ = out
	})

	t.Run("BridgeConfig", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "bridge", "config")
		_ = out
	})

	t.Run("Restart", func(t *testing.T) {
		out := runOK(t, testProjectDir, "restart")
		_ = out
	})

	time.Sleep(3 * time.Second)

	t.Run("ChainInfoAfterRestart", func(t *testing.T) {
		out := runOK(t, testProjectDir, "chain", "info")
		assertContains(t, out, "Chain ID")
	})

	t.Run("Down", func(t *testing.T) {
		out := runOK(t, testProjectDir, "down", "--volumes")
		assertContains(t, out, "stopped")
	})

	t.Run("StatusAfterDown", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "status")
		assertContains(t, out, "No Dokrypt stack running")
	})

	t.Run("DownWhenAlreadyStopped", func(t *testing.T) {
		out, _ := run(t, testProjectDir, "down")
		_ = out
	})
}

func TestE2E_Phase4_ErrorHandling(t *testing.T) {
	t.Run("UpWithoutConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := runFail(t, tmpDir, "up")
		assertContains(t, out, "dokrypt.yaml")
	})

	t.Run("DownWithoutConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		out, _ := run(t, tmpDir, "down")
		_ = out
	})

	t.Run("StatusWithoutConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		out, _ := run(t, tmpDir, "status")
		_ = out
	})

	t.Run("ChainMineWithoutStack", func(t *testing.T) {
		tmpDir := t.TempDir()
		runFail(t, tmpDir, "chain", "mine")
	})

	t.Run("SnapshotSaveWithoutStack", func(t *testing.T) {
		tmpDir := t.TempDir()
		runFail(t, tmpDir, "snapshot", "save", "test")
	})

	t.Run("ExecWithoutStack", func(t *testing.T) {
		tmpDir := t.TempDir()
		runFail(t, tmpDir, "exec", "ethereum", "echo", "hi")
	})

	t.Run("LogsWithoutStack", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, _ = run(t, tmpDir, "logs")
	})

	t.Run("InitMissingArgs", func(t *testing.T) {
		runFail(t, ".", "init")
	})

	t.Run("ChainSetBalanceMissingArgs", func(t *testing.T) {
		runFail(t, testProjectDir, "chain", "set-balance")
	})

	t.Run("ChainTimeTravelMissingArgs", func(t *testing.T) {
		runFail(t, testProjectDir, "chain", "time-travel")
	})

	t.Run("ExecMissingArgs", func(t *testing.T) {
		runFail(t, testProjectDir, "exec")
	})

	t.Run("ForkMissingStack", func(t *testing.T) {
		tmpDir := t.TempDir()
		runFail(t, tmpDir, "fork", "mainnet")
	})

	t.Run("ConfigValidateInvalidYAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "dokrypt.yaml"), []byte("{{invalid yaml"), 0o644)
		runFail(t, tmpDir, "config", "validate")
	})

	t.Run("SnapshotExportMissingArgs", func(t *testing.T) {
		runFail(t, testProjectDir, "snapshot", "export")
	})

	t.Run("SnapshotImportBadFile", func(t *testing.T) {
		badFile := filepath.Join(t.TempDir(), "bad.tar.gz")
		os.WriteFile(badFile, []byte("not a tar.gz"), 0o644)
		runFail(t, testProjectDir, "snapshot", "import", badFile)
	})
}

func TestE2E_Phase5_AllTemplates(t *testing.T) {
	templates := []struct {
		name     string
		expected []string // files that should exist
	}{
		{"evm-basic", []string{"dokrypt.yaml", "foundry.toml", "README.md"}},
		{"evm-defi", []string{"dokrypt.yaml"}},
		{"evm-nft", []string{"dokrypt.yaml"}},
		{"evm-dao", []string{"dokrypt.yaml"}},
		{"evm-token", []string{"dokrypt.yaml"}},
	}

	for _, tmpl := range templates {
		t.Run("Scaffold_"+tmpl.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, "test-"+tmpl.name)
			out := runOK(t, tmpDir, "init", projectPath, "--template", tmpl.name, "--no-git")
			assertContains(t, out, "Scaffolding")

			for _, f := range tmpl.expected {
				mustExist(t, filepath.Join(projectPath, f))
			}

			runOK(t, projectPath, "config", "validate")
		})
	}
}

func TestE2E_Phase6_PluginAndTemplateOps(t *testing.T) {
	t.Run("TemplateCreate", func(t *testing.T) {
		tmpDir := t.TempDir()
		out := runOK(t, tmpDir, "template", "create", "my-custom-template")
		assertContains(t, out, "created")

		mustExist(t, filepath.Join(tmpDir, "my-custom-template", "template.yaml"))
	})

	t.Run("PluginListEmpty", func(t *testing.T) {
		out := runOK(t, ".", "plugin", "list")
		_ = out // just verify no crash
	})

	t.Run("PluginInstallNotFound", func(t *testing.T) {
		_, _ = run(t, ".", "plugin", "install", "nonexistent-plugin-xyz")
	})
}

func TestE2E_Phase7_JSONOutput(t *testing.T) {
	t.Run("VersionJSON", func(t *testing.T) {
		out := runOK(t, ".", "version", "--json")
		assertContains(t, out, "dokrypt")
	})

	t.Run("TemplateListJSON", func(t *testing.T) {
		out := runOK(t, ".", "template", "list", "--json")
		assertContains(t, out, "evm-basic")
	})

	t.Run("DoctorJSON", func(t *testing.T) {
		out := runOK(t, ".", "doctor", "--json")
		assertContains(t, out, "Docker")
	})
}

func TestE2E_Phase8_FullCycle(t *testing.T) {
	dockerAvailable(t)

	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "cycle-test")
	runOK(t, tmpDir, "init", projectPath, "--template", "evm-basic", "--no-git")

	runOK(t, projectPath, "up", "--detach", "--timeout", "3m")
	time.Sleep(3 * time.Second)

	runOK(t, projectPath, "chain", "info")
	runOK(t, projectPath, "chain", "mine", "3")
	runOK(t, projectPath, "chain", "set-balance", "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "50000")

	runOK(t, projectPath, "snapshot", "save", "cycle-snap")
	runOK(t, projectPath, "chain", "mine", "5")
	runOK(t, projectPath, "snapshot", "restore", "cycle-snap")
	runOK(t, projectPath, "snapshot", "list")

	runOK(t, projectPath, "accounts", "list")

	runOK(t, projectPath, "down", "--volumes")

	runOK(t, projectPath, "up", "--detach", "--timeout", "3m")
	time.Sleep(3 * time.Second)

	out := runOK(t, projectPath, "chain", "info")
	assertContains(t, out, "Chain ID")

	runOK(t, projectPath, "down", "--volumes")
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}


package cli

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system requirements",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := getOutput()

			out.Info("Dokrypt Doctor")

			allPassed := true

			out.Info("Platform: %s/%s", goruntime.GOOS, goruntime.GOARCH)

			_, err := exec.LookPath("docker")
			if err != nil {
				out.Error("Docker installed — not found")
				allPassed = false
			} else {
				out.Success("Docker installed")
			}

			var dockerVersion string
			if err == nil {
				var stdout bytes.Buffer
				c := exec.Command("docker", "version", "--format", "{{.Server.APIVersion}}")
				c.Stdout = &stdout
				if runErr := c.Run(); runErr != nil {
					out.Error("Docker daemon — not running")
					allPassed = false
				} else {
					dockerVersion = strings.TrimSpace(stdout.String())
					out.Success("Docker daemon running")

					if isAPIVersionOK(dockerVersion) {
						out.Success("API version %s", dockerVersion)
					} else {
						out.Error("API version %s — minimum 1.41 required", dockerVersion)
						allPassed = false
					}
				}
			}

			freeGB, diskErr := freeDiskSpaceGB()
			if diskErr == nil {
				if freeGB >= 5 {
					out.Success("Disk space: %d GB free", freeGB)
				} else {
					out.Warning("Disk space: %d GB free (< 5 GB)", freeGB)
				}
			}

			ports := []int{8545, 5001, 8080, 4000, 8000, 6688, 3000}
			for _, port := range ports {
				if isPortAvailable(port) {
					out.Success("Port %d available", port)
				} else {
					out.Error("Port %d in use", port)
					allPassed = false
				}
			}

			if allPassed {
				out.Success("All checks passed! Ready to go.")
			} else {
				out.Warning("Some checks failed. Fix the issues above before running dokrypt up.")
			}

			return nil
		},
	}

	return cmd
}

func isPortAvailable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return true // Can't connect = port is available
	}
	conn.Close()
	return false
}

func isAPIVersionOK(version string) bool {
	parts := strings.SplitN(version, ".", 2)
	if len(parts) != 2 {
		return false
	}
	var major, minor int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	if major > 1 {
		return true
	}
	return major == 1 && minor >= 41
}

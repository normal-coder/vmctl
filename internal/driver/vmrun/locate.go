package vmrun

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

const (
	// envVMRunPath lets users override binary auto-detection.
	envVMRunPath = "VMCTL_VMRUN"
	// envVMCliPath lets users override vmcli auto-detection.
	envVMCliPath = "VMCTL_VMCLI"
	// macFusionPath is where VMware Fusion installs vmrun on macOS.
	macFusionPath = "/Applications/VMware Fusion.app/Contents/Public/vmrun"
	// macFusionCLIPath is where VMware Fusion installs vmcli on macOS
	// (ships with Fusion 13.5+).
	macFusionCLIPath = "/Applications/VMware Fusion.app/Contents/Public/vmcli"
)

// locate resolves the vmrun binary path and the matching -T host type.
// Order: explicit override > VMCTL_VMRUN env > Fusion default path > PATH.
func locate(override string) (bin, hostType string, err error) {
	if override == "" {
		override = os.Getenv(envVMRunPath)
	}

	var candidates []string
	if override != "" {
		candidates = []string{override}
	} else {
		if runtime.GOOS == "darwin" {
			candidates = append(candidates, macFusionPath)
		}
		if p, lookErr := exec.LookPath("vmrun"); lookErr == nil {
			candidates = append(candidates, p)
		}
	}

	for _, c := range candidates {
		if st, statErr := os.Stat(c); statErr == nil && !st.IsDir() {
			return c, hostTypeFor(c), nil
		}
		if p, lookErr := exec.LookPath(c); lookErr == nil {
			return p, hostTypeFor(c), nil
		}
	}
	return "", "", fmt.Errorf("vmrun not found (install VMware Fusion/Workstation or set %s)", envVMRunPath)
}

// locateVMCli resolves the vmcli binary path. Order mirrors locate:
// explicit override > VMCTL_VMCLI env > Fusion default path > PATH.
func locateVMCli(override string) (string, error) {
	if override == "" {
		override = os.Getenv(envVMCliPath)
	}

	var candidates []string
	if override != "" {
		candidates = []string{override}
	} else {
		if runtime.GOOS == "darwin" {
			candidates = append(candidates, macFusionCLIPath)
		}
		if p, lookErr := exec.LookPath("vmcli"); lookErr == nil {
			candidates = append(candidates, p)
		}
	}

	for _, c := range candidates {
		if st, statErr := os.Stat(c); statErr == nil && !st.IsDir() {
			return c, nil
		}
		if p, lookErr := exec.LookPath(c); lookErr == nil {
			return p, nil
		}
	}
	return "", errors.New(i18n.T("err.vmcli.notFound"))
}

// hostTypeFor guesses the vmrun -T value from the binary location.
func hostTypeFor(path string) string {
	if strings.Contains(path, "Fusion") {
		return "fusion"
	}
	if runtime.GOOS == "darwin" {
		// On macOS vmrun only ships with Fusion (possibly via symlink).
		return "fusion"
	}
	return "ws"
}

// absPath cleans a path for comparison purposes.
func absPath(p string) string {
	return filepath.Clean(strings.TrimSpace(p))
}

package vmrun

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// envVMRunPath lets users override binary auto-detection.
	envVMRunPath = "VMCTL_VMRUN"
	// macFusionPath is where VMware Fusion installs vmrun on macOS.
	macFusionPath = "/Applications/VMware Fusion.app/Contents/Public/vmrun"
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

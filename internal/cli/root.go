// Package cli defines vmctl's command tree.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	// Register local backends.
	_ "gitee.com/normalcoder/vmctl/internal/driver/vmrun"
)

// Version is stamped via -ldflags at build time.
var Version = "dev"

var (
	flagJSON    bool
	flagVMRun   string
	flagBackend string
)

// Execute runs the root command and maps errors to exit codes.
func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// NewRootCmd builds the vmctl command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "vmctl",
		Short:         "Control VMware virtual machines",
		Long:          "vmctl controls VMware virtual machines with a prlctl-style command interface.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.BoolVar(&flagJSON, "json", false, "output as JSON")
	pf.StringVar(&flagVMRun, "vmrun", "", "path to the vmrun binary (default: auto-detect)")
	pf.StringVar(&flagBackend, "backend", "vmrun", "backend driver to use")

	root.AddCommand(
		newListCmd(),
		newInfoCmd(),
		newStartCmd(),
		newStopCmd(),
		newSuspendCmd(),
		newResumeCmd(),
		newPauseCmd(),
		newResetCmd(),
		newCloneCmd(),
		newCreateCmd(),
		newSetCmd(),
		newDeleteCmd(),
		newVersionCmd(),
	)
	return root
}

// openDriver constructs the selected backend.
func openDriver() (driver.Driver, error) {
	return driver.Open(flagBackend, driver.Options{VMRunPath: flagVMRun})
}

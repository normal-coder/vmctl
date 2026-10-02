// Package cli defines vmctl's command tree.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
	// Register local backends.
	_ "gitee.com/normalcoder/vmctl/internal/driver/vmrun"
)

// Version is stamped via -ldflags at build time.
var Version = "dev"

var (
	flagJSON    bool
	flagVMRun   string
	flagBackend string
	flagLang    string
)

// Execute runs the root command and maps errors to exit codes.
func Execute() {
	// The command tree bakes translated help strings at build time, so
	// the language must be resolved before NewRootCmd.
	i18n.SetLang(i18n.DetectLang(os.Args))
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		err = friendlyError(err)
		fmt.Fprintln(os.Stderr, i18n.T("err.prefix")+":", err)
		os.Exit(1)
	}
}

// NewRootCmd builds the vmctl command tree.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "vmctl",
		Short:         i18n.T("root.short"),
		Long:          i18n.T("root.long"),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.BoolVar(&flagJSON, "json", false, i18n.T("flag.json"))
	pf.StringVar(&flagVMRun, "vmrun", "", i18n.T("flag.vmrun"))
	pf.StringVar(&flagBackend, "backend", "vmrun", i18n.T("flag.backend"))
	// The effective language comes from DetectLang's pre-scan; this
	// flag only makes --lang visible to cobra and to --help.
	pf.StringVar(&flagLang, "lang", "", i18n.T("flag.lang"))

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
		newSnapshotCmd(),
		newExecCmd(),
		newShellCmd(),
		newIPCmd(),
		newVersionCmd(),
	)
	return root
}

// openDriver constructs the selected backend.
func openDriver() (driver.Driver, error) {
	return driver.Open(flagBackend, driver.Options{VMRunPath: flagVMRun})
}

// friendlyError renders ErrNotSupported with the current language,
// stripping the English sentinel text the driver wraps in.
func friendlyError(err error) error {
	if !errors.Is(err, driver.ErrNotSupported) {
		return err
	}
	// Drivers wrap as fmt.Errorf("%w: <detail>", ErrNotSupported).
	detail := strings.TrimPrefix(err.Error(), driver.ErrNotSupported.Error()+": ")
	if detail == err.Error() {
		detail = ""
	}
	if detail == "" {
		return errors.New(i18n.T("err.notSupported"))
	}
	return fmt.Errorf("%s: %s", i18n.T("err.notSupported"), detail)
}

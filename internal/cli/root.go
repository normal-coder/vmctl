// Package cli defines vmctl's command tree.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"gitee.com/normalcoder/vmctl/internal/i18n"
	// Register backends.
	_ "gitee.com/normalcoder/vmctl/internal/driver/vmrun"
	_ "gitee.com/normalcoder/vmctl/internal/driver/vsphere"
)

// Version is stamped via -ldflags at build time.
var Version = "dev"

var (
	flagJSON    bool
	flagVMRun   string
	flagVMCli   string
	flagBackend string
	flagLang    string
	flagProfile string

	// rootFlags is the root persistent flag set, captured by
	// NewRootCmd so openDriver can consult Changed and tell explicit
	// CLI flags from defaults.
	rootFlags *pflag.FlagSet
)

// Execute runs the root command and maps errors to exit codes:
// exitCodeError passes its code through silently, usage errors exit 2
// with the command's usage, everything else exits 1.
func Execute() {
	// The command tree bakes translated help strings at build time, so
	// the language must be resolved before NewRootCmd.
	i18n.SetLang(i18n.DetectLang(os.Args))
	root := NewRootCmd()
	cmd, err := root.ExecuteC()
	if err == nil {
		return
	}
	plan := planExit(err)
	if plan.message != "" {
		fmt.Fprintln(os.Stderr, i18n.T("err.prefix")+":", plan.message)
		if plan.showUsage {
			fmt.Fprint(os.Stderr, cmd.UsageString())
		}
	}
	os.Exit(plan.code)
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
	pf.StringVar(&flagVMCli, "vmcli", "", i18n.T("flag.vmcli"))
	pf.StringVar(&flagBackend, "backend", "vmrun", i18n.T("flag.backend"))
	pf.StringVar(&flagProfile, "profile", "", i18n.T("flag.profile"))
	// The effective language comes from DetectLang's pre-scan; this
	// flag only makes --lang visible to cobra and to --help.
	pf.StringVar(&flagLang, "lang", "", i18n.T("flag.lang"))
	rootFlags = pf

	// Unknown or malformed flags are usage errors (exit 2).
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return usageError{err}
	})

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
	registerCompletions(root)
	localizeCompletion(root)
	return root
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// localizeHelp rewrites cobra's built-in help command in the current
// language and turns an unknown help topic into a localized usage
// error. It must run after localizeCompletion so every builtin
// command exists when the tree is walked.
func localizeHelp(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	for _, c := range root.Commands() {
		if c.Name() != "help" {
			continue
		}
		c.Short = i18n.T("cmd.help.short")
		c.Long = i18n.T("cmd.help.long")
		c.Args = helpArgs
	}
	localizeHelpFlags(root)
}

// localizeHelpFlags replaces the English usage text of every command's
// --help flag. InitDefaultHelpFlag creates the flag first; cobra only
// fills it in when missing, so execute-time re-entry keeps our text.
func localizeHelpFlags(c *cobra.Command) {
	c.InitDefaultHelpFlag()
	if f := c.Flags().Lookup("help"); f != nil {
		f.Usage = fmt.Sprintf(i18n.T("flag.help"), c.Name())
	}
	for _, sub := range c.Commands() {
		localizeHelpFlags(sub)
	}
}

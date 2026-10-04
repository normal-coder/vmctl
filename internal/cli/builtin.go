package cli

import (
	"fmt"
	"strings"

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

// localizeFlagDefaults rewrites pflag's English " (default ...)"
// flag-usage suffix in the current language. Registered once via
// cobra.AddTemplateFunc and applied inside usageTemplate's pipeline;
// en is a no-op because the replacement text equals the original.
func localizeFlagDefaults(s string) string {
	return strings.ReplaceAll(s, " (default ", i18n.T("flag.defaultWord"))
}

// usageTemplate is cobra's defaultUsageTemplate with each section
// label localized. SetUsageTemplate cannot add template funcs, so the
// labels are interpolated here with plain string concatenation —
// cobra template actions ({{...}}) stay verbatim. The usage template
// is set once on root; every subcommand inherits it.
func usageTemplate() string {
	return i18n.T("usage.usage") + `:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

` + i18n.T("usage.aliases") + `:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

` + i18n.T("usage.examples") + `:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

` + i18n.T("usage.availableCommands") + `:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

` + i18n.T("usage.additionalCommands") + `:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + i18n.T("usage.flags") + `:
{{.LocalFlags.FlagUsages | localizeFlagDefaults | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + i18n.T("usage.globalFlags") + `:
{{.InheritedFlags.FlagUsages | localizeFlagDefaults | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

` + i18n.T("usage.additionalHelp") + `:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

` + fmt.Sprintf(i18n.T("usage.moreInfo"), "{{.CommandPath}}") + `{{end}}
`
}

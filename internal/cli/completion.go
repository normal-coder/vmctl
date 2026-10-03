package cli

import (
	"context"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/config"
	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// vmNameList resolves the VM display names offered for reference
// completion. Package-level so tests can stub it without a live
// backend.
var vmNameList = func(ctx context.Context) ([]string, error) {
	d, err := openDriver()
	if err != nil {
		return nil, err
	}
	vms, err := d.List(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(vms))
	for _, vm := range vms {
		names = append(names, vm.Name)
	}
	return names, nil
}

// vmRefCompletion completes the first positional argument with VM
// names. Completion must never fail loudly: any backend error (e.g.
// vsphere offline) yields no candidates instead of an error line.
func vmRefCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := vmNameList(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasPrefix(n, toComplete) {
			out = append(out, n)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeBackend offers every registered backend driver.
func completeBackend(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	names := driver.Available()
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeLang offers the supported interface languages.
func completeLang(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"zh", "en"}, cobra.ShellCompDirectiveNoFileComp
}

// completeProfile offers configured profile names; an unreadable
// config simply yields no candidates.
func completeProfile(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	path, mustExist := config.Path()
	cfg, err := config.Load(path, mustExist)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cfg.Names(), cobra.ShellCompDirectiveNoFileComp
}

// registerCompletions wires flag value and positional completion onto
// the command tree.
func registerCompletions(root *cobra.Command) {
	_ = root.RegisterFlagCompletionFunc("backend", completeBackend)
	_ = root.RegisterFlagCompletionFunc("lang", completeLang)
	_ = root.RegisterFlagCompletionFunc("profile", completeProfile)

	for _, c := range root.Commands() {
		switch c.Name() {
		case "info", "start", "stop", "suspend", "resume", "pause",
			"reset", "clone", "set", "delete", "exec", "shell", "ip":
			c.ValidArgsFunction = vmRefCompletion
		case "create":
			// The positional arg is a new name; only --from takes a VM ref.
			_ = c.RegisterFlagCompletionFunc("from", vmRefCompletion)
		case "snapshot":
			// All four subcommands take the VM ref first.
			for _, sub := range c.Commands() {
				sub.ValidArgsFunction = vmRefCompletion
			}
		}
	}
}

// localizeCompletion rewrites cobra's built-in completion command
// help in the current language. cobra bakes English at build time.
func localizeCompletion(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	c, _, err := root.Find([]string{"completion"})
	if err != nil || c.Name() != "completion" {
		return
	}
	c.Short = i18n.T("cmd.completion.short")
	c.Long = i18n.T("cmd.completion.long")
	// Runnable + Args so stray arguments become usage errors instead
	// of silently printing help (cobra skips Args on non-runnables).
	c.Args = noArgs
	c.Run = func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	}
	for _, sub := range c.Commands() {
		switch sub.Name() {
		case "bash", "zsh", "fish", "powershell":
			sub.Short = i18n.T("cmd.completion." + sub.Name())
			sub.Args = noArgs
		}
	}
}

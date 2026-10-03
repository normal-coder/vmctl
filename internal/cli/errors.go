package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// usageError marks a command-line usage problem (unknown flag, wrong
// argument count). Execute prints the message plus the command's
// usage and exits with code 2.
type usageError struct{ error }

// exitCodeError carries a program exit code out of a RunE without
// printing anything — e.g. a guest command's own exit status being
// mirrored by `vmctl exec`.
type exitCodeError int

func (e exitCodeError) Error() string { return "exit code" }

// exactArgs requires exactly n positional arguments. The wrappers are
// implemented locally rather than wrapping cobra's validators so the
// message is localized directly — cobra's English text never leaks.
func exactArgs(n int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != n {
			return usageError{fmt.Errorf(i18n.T("err.args.exact"), n, len(args))}
		}
		return nil
	}
}

// minimumNArgs requires at least n positional arguments.
func minimumNArgs(n int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) < n {
			return usageError{fmt.Errorf(i18n.T("err.args.minimum"), n, len(args))}
		}
		return nil
	}
}

// noArgs rejects any positional argument.
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return usageError{fmt.Errorf(i18n.T("err.unknownCommand"), args[0], cmd.CommandPath())}
	}
	return nil
}

// requireFlags reports missing required flags as a usage error,
// running during ValidateArgs — before cobra's own English
// ValidateRequiredFlags check (which has no localization hook).
func requireFlags(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, _ []string) error {
		var missing []string
		for _, n := range names {
			if !cmd.Flags().Changed(n) {
				missing = append(missing, "--"+n)
			}
		}
		if len(missing) == 0 {
			return nil
		}
		return usageError{fmt.Errorf(i18n.T("err.flag.required"), strings.Join(missing, ", "))}
	}
}

// flagError translates pflag's typed parse errors into localized
// usage errors. pflag passes its concrete types through ParseFlags
// untouched, so the getters recover flag names without matching any
// English message text; unrecognized errors pass through as-is.
func flagError(_ *cobra.Command, err error) error {
	if errors.Is(err, pflag.ErrHelp) {
		return err
	}
	var ne *pflag.NotExistError
	var vr *pflag.ValueRequiredError
	var syn *pflag.InvalidSyntaxError
	var inv *pflag.InvalidValueError
	switch {
	case errors.As(err, &ne):
		if ne.GetSpecifiedShortnames() != "" {
			return usageError{fmt.Errorf(i18n.T("err.flag.unknownShort"), ne.GetSpecifiedShortnames())}
		}
		return usageError{fmt.Errorf(i18n.T("err.flag.unknownLong"), ne.GetSpecifiedName())}
	case errors.As(err, &vr):
		if vr.GetSpecifiedShortnames() != "" {
			return usageError{fmt.Errorf(i18n.T("err.flag.needsValueShort"), vr.GetSpecifiedShortnames())}
		}
		return usageError{fmt.Errorf(i18n.T("err.flag.needsValue"), vr.GetSpecifiedName())}
	case errors.As(err, &syn):
		return usageError{fmt.Errorf(i18n.T("err.flag.badSyntax"), syn.GetSpecifiedFlag())}
	case errors.As(err, &inv):
		name := ""
		if f := inv.GetFlag(); f != nil {
			name = f.Name
		}
		// The unwrapped cause is an English strconv error; drop it.
		return usageError{fmt.Errorf(i18n.T("err.flag.invalidValue"), name, inv.GetValue())}
	default:
		return usageError{err}
	}
}

// rootArgs rejects unknown subcommands. Once root.Args is set, cobra's
// Find stops consulting legacyArgs, so this is the single place a
// misspelled top-level command is caught — as a localized usage error.
func rootArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return usageError{unknownCommandErr(cmd, args[0])}
}

// unknownCommandErr formats a localized "unknown command" error,
// appending did-you-mean suggestions when cobra's matcher finds any.
func unknownCommandErr(cmd *cobra.Command, arg string) error {
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2 // same guard cobra applies internally
	}
	sugg := cmd.SuggestionsFor(arg)
	if len(sugg) == 0 {
		return fmt.Errorf(i18n.T("err.unknownCommand"), arg, cmd.CommandPath())
	}
	return fmt.Errorf(i18n.T("err.unknownCommandSuggest"), arg, cmd.CommandPath(),
		strings.Join(sugg, "\n  "))
}

// helpArgs mirrors the built-in help command's Find lookup: a topic
// that falls back to the root (and isn't the root itself) is unknown.
// Without this, setting root.Args would make cobra silently print the
// root help for `vmctl help nope` instead of reporting the miss.
func helpArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	root := cmd.Root()
	found, _, err := root.Find(args)
	if err != nil ||
		(found == root && args[0] != root.Name() && !root.HasAlias(args[0])) {
		return usageError{fmt.Errorf(i18n.T("err.help.unknownTopic"), args[0])}
	}
	return nil
}

// exitPlan describes how Execute turns an error into process exit.
type exitPlan struct {
	code      int
	message   string // "" prints nothing (exitCodeError passthrough)
	showUsage bool
}

// planExit classifies a command error: a carried exit code (silent),
// a usage error (code 2 + usage text) or a normal failure (code 1).
func planExit(err error) exitPlan {
	var ec exitCodeError
	if errors.As(err, &ec) {
		return exitPlan{code: int(ec)}
	}
	var ue usageError
	if errors.As(err, &ue) {
		return exitPlan{code: 2, message: ue.Error(), showUsage: true}
	}
	return exitPlan{code: 1, message: friendlyError(err).Error()}
}

// friendlyError strips the English sentinel text drivers wrap in,
// leaving the already-localized detail sentence. ErrNotFound's detail
// is a full sentence in the current language since M5.
func friendlyError(err error) error {
	type strip struct {
		sentinel error
		label    string // localized label, "" = detail only
	}
	for _, s := range []strip{
		{driver.ErrNotSupported, i18n.T("err.notSupported")},
		{driver.ErrNotFound, ""},
	} {
		if !errors.Is(err, s.sentinel) {
			continue
		}
		detail := strings.TrimPrefix(err.Error(), s.sentinel.Error()+": ")
		if detail == err.Error() {
			detail = ""
		}
		switch {
		case s.label == "" && detail == "":
			return err
		case s.label == "":
			return errors.New(detail)
		case detail == "":
			return errors.New(s.label)
		default:
			return errors.New(s.label + ": " + detail)
		}
	}
	return err
}

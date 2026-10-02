package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

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

// exactArgs wraps cobra.ExactArgs so argument-count mistakes become
// usage errors.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return usageError{err}
		}
		return nil
	}
}

// minimumNArgs wraps cobra.MinimumNArgs as a usage error.
func minimumNArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(n)(cmd, args); err != nil {
			return usageError{err}
		}
		return nil
	}
}

// noArgs wraps cobra.NoArgs as a usage error.
func noArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return usageError{err}
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

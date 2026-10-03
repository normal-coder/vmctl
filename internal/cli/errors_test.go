package cli

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// TestPlanExitCodes pins the exit-code priority table:
// carried code > usage (2) > not found (3) > not supported (4) > 1.
func TestPlanExitCodes(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		code       int
		message    string // substring of plan.message; "" = no content check
		wantSilent bool
		showUsage  bool
	}{
		{"carried", exitCodeError(7), 7, "", true, false},
		{"usage", usageError{errors.New("unknown flag: --bogus")}, 2, "--bogus", false, true},
		{"usageOverNotFound", usageError{driver.WrapNotFound("找不到虚拟机 \"nope\"")}, 2, "", false, true},
		{"notFound", driver.WrapNotFound("找不到虚拟机 \"nope\"（可用：a）"), 3, "nope", false, false},
		{"notSupported", fmt.Errorf("%w: 因为 FFF", driver.ErrNotSupported), 4, i18n.T("err.notSupported"), false, false},
		{"normal", errors.New("boom"), 1, "boom", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan := planExit(c.err)
			if plan.code != c.code {
				t.Errorf("code = %d, want %d", plan.code, c.code)
			}
			if plan.showUsage != c.showUsage {
				t.Errorf("showUsage = %v, want %v", plan.showUsage, c.showUsage)
			}
			if c.wantSilent {
				if plan.message != "" {
					t.Errorf("message = %q, want silent", plan.message)
				}
				return
			}
			if c.message != "" && !strings.Contains(plan.message, c.message) {
				t.Errorf("message = %q, want it to contain %q", plan.message, c.message)
			}
		})
	}

	// The English sentinel text must never leak into 3/4 messages.
	if got := planExit(driver.WrapNotFound("找不到虚拟机 \"nope\"")); strings.Contains(got.message, driver.ErrNotFound.Error()) {
		t.Errorf("notFound message leaked the sentinel: %q", got.message)
	}
	if got := planExit(fmt.Errorf("%w: x", driver.ErrNotSupported)); strings.Contains(got.message, driver.ErrNotSupported.Error()) {
		t.Errorf("notSupported message leaked the sentinel: %q", got.message)
	}
}

func TestFriendlyErrorStripsSentinel(t *testing.T) {
	t.Run("notFound", func(t *testing.T) {
		err := driver.WrapNotFound("找不到虚拟机 \"nope\"（可用：a）")
		got := friendlyError(err).Error()
		if strings.Contains(got, "virtual machine not found") {
			t.Errorf("English sentinel leaked: %q", got)
		}
		if !strings.Contains(got, "nope") {
			t.Errorf("detail lost: %q", got)
		}
	})
	t.Run("notSupported", func(t *testing.T) {
		err := fmt.Errorf("%w: %s", driver.ErrNotSupported, "因为 FFF")
		got := friendlyError(err).Error()
		if strings.Contains(got, driver.ErrNotSupported.Error()) {
			t.Errorf("English sentinel leaked: %q", got)
		}
		if !strings.Contains(got, i18n.T("err.notSupported")) || !strings.Contains(got, "因为 FFF") {
			t.Errorf("label/detail lost: %q", got)
		}
	})
	t.Run("unrelated passes through", func(t *testing.T) {
		err := errors.New("plain")
		if got := friendlyError(err); got.Error() != "plain" {
			t.Errorf("friendlyError(plain) = %q", got)
		}
	})
}

func TestArgValidatorsAreUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args cobra.PositionalArgs
		in   []string
		want string
	}{
		{"exact", exactArgs(1), nil, fmt.Sprintf(i18n.T("err.args.exact"), 1, 0)},
		{"exactExtra", exactArgs(1), []string{"a", "b"}, fmt.Sprintf(i18n.T("err.args.exact"), 1, 2)},
		{"minimum", minimumNArgs(2), []string{"one"}, fmt.Sprintf(i18n.T("err.args.minimum"), 2, 1)},
		{"noArgs", noArgs, []string{"extra"}, fmt.Sprintf(i18n.T("err.unknownCommand"), "extra", "version")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "version"}
			err := c.args(cmd, c.in)
			if err == nil {
				t.Fatal("want error")
			}
			var ue usageError
			if !errors.As(err, &ue) {
				t.Errorf("err = %v, want usageError", err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("message = %q, want it to contain %q", err.Error(), c.want)
			}
		})
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"--bogus"})
	err := root.Execute()
	if err == nil {
		t.Fatal("unknown flag must error")
	}
	var ue usageError
	if !errors.As(err, &ue) {
		t.Errorf("err = %v (%T), want usageError", err, err)
	}
}

// TestFlagErrorPathsAreUsageErrors drives real pflag parse failures
// through Execute — pflag's error fields are unexported, so the only
// way to produce them is actual flag parsing.
func TestFlagErrorPathsAreUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"long", []string{"--bogus"}, fmt.Sprintf(i18n.T("err.flag.unknownLong"), "bogus")},
		{"short", []string{"-Z"}, fmt.Sprintf(i18n.T("err.flag.unknownShort"), "Z")},
		{"longNeedsValue", []string{"--backend"}, fmt.Sprintf(i18n.T("err.flag.needsValue"), "backend")},
		{"shortNeedsValue", []string{"clone", "demo", "-n"}, fmt.Sprintf(i18n.T("err.flag.needsValueShort"), "n")},
		{"badSyntax", []string{"---bad"}, fmt.Sprintf(i18n.T("err.flag.badSyntax"), "---bad")},
		{"invalidValue", []string{"--json=maybe"}, fmt.Sprintf(i18n.T("err.flag.invalidValue"), "json", "maybe")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := NewRootCmd()
			root.SetArgs(c.args)
			err := root.Execute()
			if err == nil {
				t.Fatal("want error")
			}
			var ue usageError
			if !errors.As(err, &ue) {
				t.Errorf("err = %v (%T), want usageError", err, err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("message = %q, want it to contain %q", err.Error(), c.want)
			}
		})
	}
}

// TestFlagErrorEnglish runs a subset under the English catalog to
// prove the localized templates switch language too.
func TestFlagErrorEnglish(t *testing.T) {
	i18n.SetLang("en")
	defer i18n.SetLang("zh")

	root := NewRootCmd()
	root.SetArgs([]string{"--bogus"})
	err := root.Execute()
	want := fmt.Sprintf(i18n.T("err.flag.unknownLong"), "bogus")
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want to contain %q", err, want)
	}
}

// TestRequiredFlagIsUsageError: clone/create required flags are
// rejected during ValidateArgs, before cobra's English
// ValidateRequiredFlags can run.
func TestRequiredFlagIsUsageError(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"clone", "demo"}, fmt.Sprintf(i18n.T("err.flag.required"), "--name")},
		{[]string{"create", "foo"}, fmt.Sprintf(i18n.T("err.flag.required"), "--from")},
	}
	for _, c := range cases {
		root := NewRootCmd()
		root.SetArgs(c.args)
		err := root.Execute()
		if err == nil {
			t.Fatalf("%v: want error", c.args)
		}
		var ue usageError
		if !errors.As(err, &ue) {
			t.Errorf("%v: err = %v, want usageError", c.args, err)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: message = %q, want it to contain %q", c.args, err.Error(), c.want)
		}
	}

	// With the required flag present the usage check passes; the
	// command then fails in the backend (an unresolvable VM name)
	// with a non-usage error — asserting that requires no driver.
	root := NewRootCmd()
	root.SetArgs([]string{"clone", "no-such-vm-xyz", "--name", "copy"})
	err := root.Execute()
	if err == nil {
		t.Fatal("want backend error for missing VM")
	}
	var ue usageError
	if errors.As(err, &ue) {
		t.Errorf("required flag satisfied, must not be a usage error: %v", err)
	}
}

// --- unknown subcommand / built-in commands (stage B) ---

func TestUnknownSubcommandSuggests(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"lst"})
	err := root.Execute()
	if err == nil {
		t.Fatal("want error")
	}
	var ue usageError
	if !errors.As(err, &ue) {
		t.Errorf("err = %v, want usageError", err)
	}
	want := fmt.Sprintf(i18n.T("err.unknownCommandSuggest"), "lst", "vmctl", "list")
	if !strings.Contains(err.Error(), want) {
		t.Errorf("message = %q, want it to contain %q", err.Error(), want)
	}
}

func TestUnknownSubcommandNoSuggest(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"zzzzzz"})
	err := root.Execute()
	if err == nil {
		t.Fatal("want error")
	}
	var ue usageError
	if !errors.As(err, &ue) {
		t.Errorf("err = %v, want usageError", err)
	}
	want := fmt.Sprintf(i18n.T("err.unknownCommand"), "zzzzzz", "vmctl")
	if !strings.Contains(err.Error(), want) {
		t.Errorf("message = %q, want it to contain %q", err.Error(), want)
	}
}

func TestBareRootPrintsHelp(t *testing.T) {
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{})
	if err := root.Execute(); err != nil {
		t.Fatalf("bare root must print help and succeed: %v", err)
	}
	if !strings.Contains(buf.String(), "vmctl") {
		t.Errorf("stdout = %q, want the help text", buf.String())
	}
}

func TestSnapshotBogusArg(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"snapshot", "nope"})
	err := root.Execute()
	if err == nil {
		t.Fatal("stray snapshot argument must error")
	}
	var ue usageError
	if !errors.As(err, &ue) {
		t.Errorf("err = %v, want usageError", err)
	}
	want := fmt.Sprintf(i18n.T("err.unknownCommand"), "nope", "vmctl snapshot")
	if !strings.Contains(err.Error(), want) {
		t.Errorf("message = %q, want it to contain %q", err.Error(), want)
	}
}

func TestHelpUnknownTopic(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		root := NewRootCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"help", "nope"})
		err := root.Execute()
		if err == nil {
			t.Fatal("unknown help topic must error (not silently print root help)")
		}
		var ue usageError
		if !errors.As(err, &ue) {
			t.Errorf("err = %v, want usageError", err)
		}
		want := fmt.Sprintf(i18n.T("err.help.unknownTopic"), "nope")
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message = %q, want it to contain %q", err.Error(), want)
		}
	})
	t.Run("known topic", func(t *testing.T) {
		root := NewRootCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetArgs([]string{"help", "list"})
		if err := root.Execute(); err != nil {
			t.Fatalf("help list: %v", err)
		}
		if !strings.Contains(buf.String(), i18n.T("cmd.list.short")) {
			t.Errorf("stdout = %q, want list help", buf.String())
		}
	})
	t.Run("bare help", func(t *testing.T) {
		root := NewRootCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetArgs([]string{"help"})
		if err := root.Execute(); err != nil {
			t.Fatalf("bare help: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("bare help must print usage")
		}
	})
}

func TestHelpLocalized(t *testing.T) {
	for _, lang := range []string{i18n.ZH, i18n.EN} {
		t.Run(lang, func(t *testing.T) {
			i18n.SetLang(lang)
			t.Cleanup(func() { i18n.SetLang(i18n.ZH) })

			root := NewRootCmd()
			for _, cmd := range root.Commands() {
				if cmd.Name() != "help" {
					continue
				}
				if cmd.Short != i18n.T("cmd.help.short") {
					t.Errorf("help short = %q, want %q", cmd.Short, i18n.T("cmd.help.short"))
				}
				return
			}
			t.Fatal("help command not registered")
		})
	}
}

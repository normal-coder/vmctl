package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

func TestPlanExitCarriedCodeSilent(t *testing.T) {
	plan := planExit(exitCodeError(7))
	if plan.code != 7 || plan.message != "" || plan.showUsage {
		t.Errorf("planExit(exitCodeError(7)) = %+v, want code 7 silent", plan)
	}
}

func TestPlanExitUsageError(t *testing.T) {
	plan := planExit(usageError{errors.New("unknown flag: --bogus")})
	if plan.code != 2 || !plan.showUsage {
		t.Errorf("planExit(usageError) = %+v, want code 2 with usage", plan)
	}
	if !strings.Contains(plan.message, "--bogus") {
		t.Errorf("message = %q, want the flag error text", plan.message)
	}
}

func TestPlanExitNormalError(t *testing.T) {
	plan := planExit(errors.New("boom"))
	if plan.code != 1 || plan.showUsage || plan.message == "" {
		t.Errorf("planExit(boom) = %+v, want code 1 with message", plan)
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
	}{
		{"exact", exactArgs(1), nil},
		{"minimum", minimumNArgs(2), []string{"one"}},
		{"noArgs", noArgs, []string{"extra"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.args(&cobra.Command{}, c.in)
			if err == nil {
				t.Fatal("want error")
			}
			var ue usageError
			if !errors.As(err, &ue) {
				t.Errorf("err = %v, want usageError", err)
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

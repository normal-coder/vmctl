package cli

import (
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// The command tree bakes translated strings when it is built, so each
// language must be exercised against a freshly constructed tree.
func TestHelpLanguage(t *testing.T) {
	cases := []struct {
		lang string
		want string
		not  string
	}{
		{i18n.ZH, "列出虚拟机", "List virtual machines"},
		{i18n.EN, "List virtual machines", "列出虚拟机"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			i18n.SetLang(c.lang)
			t.Cleanup(func() { i18n.SetLang(i18n.ZH) })

			root := NewRootCmd()
			short := ""
			for _, cmd := range root.Commands() {
				if cmd.Name() == "list" {
					short = cmd.Short
				}
			}
			if short == "" {
				t.Fatal("list command not found")
			}
			if !strings.Contains(short, c.want) {
				t.Errorf("list short = %q, want containing %q", short, c.want)
			}
			if strings.Contains(short, c.not) {
				t.Errorf("list short = %q, must not contain %q", short, c.not)
			}
		})
	}
}

// The root --lang flag must be registered so it shows up in help.
func TestLangFlagRegistered(t *testing.T) {
	root := NewRootCmd()
	if f := root.PersistentFlags().Lookup("lang"); f == nil {
		t.Fatal("--lang flag not registered")
	}
}

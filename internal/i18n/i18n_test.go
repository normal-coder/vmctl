package i18n

import (
	"testing"
)

// TestCatalogParity enforces that zh and en define the same keys, so
// a message added in one language cannot silently go untranslated.
func TestCatalogParity(t *testing.T) {
	for key := range zhCatalog {
		if _, ok := enCatalog[key]; !ok {
			t.Errorf("key %q missing from enCatalog", key)
		}
	}
	for key := range enCatalog {
		if _, ok := zhCatalog[key]; !ok {
			t.Errorf("key %q missing from zhCatalog", key)
		}
	}
	if len(zhCatalog) == 0 {
		t.Fatal("catalog is empty")
	}
}

func TestT(t *testing.T) {
	t.Cleanup(func() { SetLang(ZH) })

	SetLang(ZH)
	if got := T("cmd.list.short"); got != "列出虚拟机" {
		t.Errorf("zh: %q", got)
	}
	SetLang(EN)
	if got := T("cmd.list.short"); got != "List virtual machines" {
		t.Errorf("en: %q", got)
	}

	// Sprintf interpolation.
	SetLang(EN)
	if got := T("err.shell.args", 3); got != "accepts 1 arg(s), received 3" {
		t.Errorf("interp: %q", got)
	}
	SetLang(ZH)
	if got := T("err.shell.args", 3); got != "只接受 1 个参数，收到 3 个" {
		t.Errorf("interp zh: %q", got)
	}

	// Unknown key falls back to the key itself.
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key: %q", got)
	}
}

func TestSetLangUnsupported(t *testing.T) {
	t.Cleanup(func() { SetLang(ZH) })
	SetLang("fr")
	if Lang() != ZH {
		t.Errorf("unsupported language should fall back to zh, got %q", Lang())
	}
}

func TestDetectLang(t *testing.T) {
	cases := []struct {
		name string
		args []string
		env  string
		want string
	}{
		{"default", []string{"vmctl", "list"}, "", ZH},
		{"flag equals", []string{"vmctl", "--lang=en", "list"}, "", EN},
		{"flag space", []string{"vmctl", "--lang", "en", "list"}, "", EN},
		{"flag wins over env", []string{"vmctl", "--lang=zh"}, "en", ZH},
		{"env", []string{"vmctl"}, "en", EN},
		{"env zh-CN", []string{"vmctl"}, "zh-CN.UTF-8", ZH},
		{"invalid flag falls through to env", []string{"vmctl", "--lang=fr"}, "en", EN},
		{"invalid env defaults zh", []string{"vmctl"}, "fr", ZH},
		{"after dash ignored", []string{"vmctl", "shell", "vm", "--", "--lang=en"}, "", ZH},
		{"malformed flag alone", []string{"vmctl", "--lang"}, "", ZH},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("VMCTL_LANG", c.env) // empty string behaves as unset
			if got := DetectLang(c.args); got != c.want {
				t.Errorf("DetectLang(%v) env=%q = %q, want %q", c.args, c.env, got, c.want)
			}
		})
	}
}

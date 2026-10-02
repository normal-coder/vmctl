// Package i18n provides message catalogs for user-facing strings.
//
// Language selection: --lang flag (pre-scanned from os.Args before the
// command tree is built) > VMCTL_LANG env > zh. The system LANG is
// deliberately not consulted so an en_US shell cannot lock a Chinese
// user into English output.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Supported languages.
const (
	ZH = "zh"
	EN = "en"
)

var (
	mu   sync.RWMutex
	lang = ZH
)

// catalogs holds every message directory, keyed by language.
var catalogs = map[string]map[string]string{
	ZH: zhCatalog,
	EN: enCatalog,
}

// SetLang switches the active language. Unsupported values fall back
// to zh so T never fails.
func SetLang(l string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := catalogs[l]; ok {
		lang = l
		return
	}
	lang = ZH
}

// Lang returns the active language.
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return lang
}

// T looks up key in the active catalog, falling back to en and then
// to the key itself (so missing messages stay visible in output).
// With args it acts as fmt.Sprintf.
func T(key string, args ...any) string {
	mu.RLock()
	l := lang
	mu.RUnlock()

	msg, ok := catalogs[l][key]
	if !ok {
		msg, ok = catalogs[EN][key]
		if !ok {
			msg = key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

// DetectLang resolves the message language from CLI args and the
// environment. It understands `--lang=xx` and `--lang xx`; scanning
// stops at `--` so trailing ssh passthrough args are ignored.
func DetectLang(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if v, ok := strings.CutPrefix(a, "--lang="); ok {
			if l, ok := normalize(v); ok {
				return l
			}
			continue
		}
		if a == "--lang" && i+1 < len(args) {
			if l, ok := normalize(args[i+1]); ok {
				return l
			}
			i++
		}
	}
	if v := os.Getenv("VMCTL_LANG"); v != "" {
		if l, ok := normalize(v); ok {
			return l
		}
	}
	return ZH
}

// normalize maps values like "zh-CN" or "en_US.UTF-8" onto zh/en.
func normalize(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	switch {
	case strings.HasPrefix(v, "zh"):
		return ZH, true
	case strings.HasPrefix(v, "en"):
		return EN, true
	}
	return "", false
}

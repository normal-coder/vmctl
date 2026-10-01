// Package output renders VMCTL results as tables or JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"gitee.com/normalcoder/vmctl/internal/model"
)

// PrintVMs renders a VM list. JSON output is a machine-readable array.
func PrintVMs(w io.Writer, vms []model.VM, asJSON bool) error {
	if asJSON {
		return writeJSON(w, vms)
	}
	if len(vms) == 0 {
		_, err := fmt.Fprintln(w, "No virtual machines found.")
		return err
	}
	t := newTable()
	t.AppendHeader(table.Row{"NAME", "STATE", "CPUS", "MEM", "GUEST OS", "UUID"})
	for _, vm := range vms {
		t.AppendRow(table.Row{
			vm.Name,
			vm.State,
			intOrDash(vm.CPUs),
			memOrDash(vm.MemoryMB),
			valueOrDash(vm.GuestOS),
			valueOrDash(vm.ID),
		})
	}
	_, err := fmt.Fprintln(w, t.Render())
	return err
}

// PrintInfo renders a single VM's details.
func PrintInfo(w io.Writer, info *model.VMInfo, asJSON bool) error {
	if asJSON {
		return writeJSON(w, info)
	}
	t := newTable()
	t.AppendHeader(table.Row{"FIELD", "VALUE"})
	rows := []table.Row{
		{"Name", info.Name},
		{"State", info.State},
		{"UUID", valueOrDash(info.ID)},
		{"Guest OS", valueOrDash(info.GuestOS)},
		{"CPUs", intOrDash(info.CPUs)},
		{"Memory", memOrDash(info.MemoryMB)},
		{"Tools", valueOrDash(info.ToolsState)},
		{"Guest IP", valueOrDash(info.GuestIP)},
		{"Path", valueOrDash(info.Path)},
	}
	for _, r := range rows {
		t.AppendRow(r)
	}
	_, err := fmt.Fprintln(w, t.Render())
	return err
}

// PrintOK prints a short confirmation for mutating commands.
func PrintOK(w io.Writer, action, name string, asJSON bool) error {
	if asJSON {
		return writeJSON(w, map[string]string{"action": action, "vm": name, "result": "ok"})
	}
	_, err := fmt.Fprintf(w, "%s %s: ok\n", action, name)
	return err
}

// PrintCreated confirms a command that produced a new VM, including
// the path it was created at.
func PrintCreated(w io.Writer, action, name, path string, asJSON bool) error {
	if asJSON {
		return writeJSON(w, map[string]string{"action": action, "vm": name, "result": "ok", "path": path})
	}
	_, err := fmt.Fprintf(w, "%s %s: ok\n  -> %s\n", action, name, path)
	return err
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newTable() table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.Style().Format.Header = text.FormatUpper
	return t
}

func intOrDash(v int) string {
	if v == 0 {
		return "-"
	}
	return strconv.Itoa(v)
}

func memOrDash(mb int) string {
	if mb == 0 {
		return "-"
	}
	return strconv.Itoa(mb) + " MB"
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

package vmrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/model"
)

// --- fixtures ---------------------------------------------------------

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	return string(b)
}

// fakeExec records every vmrun invocation and serves scripted responses.
type fakeExec struct {
	calls   [][]string
	respond func(args []string) (string, error)
}

func (f *fakeExec) exec(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{}, args...))
	out, err := f.respond(args)
	return []byte(out), err
}

// stripHostType removes the leading "-T <hostType>" runner prefix.
func stripHostType(args []string) []string {
	if len(args) >= 2 && args[0] == "-T" {
		return args[2:]
	}
	return args
}

// lastArgs returns the argument list of the last invocation (without -T).
func (f *fakeExec) lastArgs() []string {
	if len(f.calls) == 0 {
		return nil
	}
	return stripHostType(f.calls[len(f.calls)-1])
}

// findArgs returns the args of the most recent call whose command
// matches cmd, or nil. It tolerates the async start goroutine.
func (f *fakeExec) findArgs(cmd string) []string {
	for i := len(f.calls) - 1; i >= 0; i-- {
		if a := stripHostType(f.calls[i]); len(a) > 0 && a[0] == cmd {
			return a
		}
	}
	return nil
}

// waitArgsLike polls until a call exactly matching want appears
// (needed because start runs in a background goroutine).
func (f *fakeExec) waitArgsLike(want []string) []string {
	wantStr := strings.Join(want, " ")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, c := range f.calls {
			if a := stripHostType(c); strings.Join(a, " ") == wantStr {
				return a
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// newTestEnv builds a temp inventory + vmx files and a driver wired to
// the fake exec. It returns the driver, the fake and a name→vmx map.
func newTestEnv(t *testing.T) (*Driver, *fakeExec, map[string]string) {
	t.Helper()
	dir := t.TempDir()

	paths := map[string]string{}
	for _, name := range []string{"demo-one", "demo-two"} {
		vmx := filepath.Join(dir, name+".vmwarevm", name+".vmx")
		if err := os.MkdirAll(filepath.Dir(vmx), 0o755); err != nil {
			t.Fatal(err)
		}
		content := strings.ReplaceAll(readTestdata(t, "demo.vmx"), "demo-one", name)
		if err := os.WriteFile(vmx, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths[name] = vmx
	}

	inv := filepath.Join(dir, "vmInventory")
	invContent := strings.Join([]string{
		`.encoding = "UTF-8"`,
		`vmlist2.config = "` + paths["demo-one"] + `"`,
		`vmlist2.DisplayName = "demo-one"`,
		`vmlist2.UUID = "56 4d 11 56 b0 39 fa a4-76 88 de e2 74 45 cc cd"`,
		`vmlist4.config = "` + paths["demo-two"] + `"`,
		`vmlist4.DisplayName = "demo-two"`,
		`vmlist4.UUID = "56 4d ef 6c ba f4 4c d9-7d 9f 4a 1f 2a 67 87 c8"`,
		`vmlist1.config = ""`,
		`index.count = "2"`,
	}, "\n")
	if err := os.WriteFile(inv, []byte(invContent), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeExec{
		respond: func(args []string) (string, error) {
			cmd := stripHostType(args)
			if len(cmd) == 0 {
				return "", nil
			}
			switch cmd[0] {
			case "list":
				return "Total running VMs: 1\n" + paths["demo-one"] + "\n", nil
			case "clone":
				// Emulate vmrun: create the destination bundle + vmx
				// and apply -cloneName to the destination displayName.
				src, dest := cmd[1], cmd[2]
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return "", err
				}
				data, err := os.ReadFile(src)
				if err != nil {
					return "", err
				}
				if err := os.WriteFile(dest, data, 0o644); err != nil {
					return "", err
				}
				for _, a := range cmd {
					if cn, ok := strings.CutPrefix(a, "-cloneName="); ok {
						if err := setVMXKeys(dest, map[string]string{"displayName": cn}); err != nil {
							return "", err
						}
					}
				}
				return "", nil
			case "deleteVM":
				// Emulate vmrun: remove the whole bundle.
				return "", os.RemoveAll(filepath.Dir(cmd[1]))
			}
			return "", nil
		},
	}
	d := &Driver{
		runner:  &runner{bin: "/fake/vmrun", hostType: "fusion", execFn: fake.exec},
		invPath: inv,
	}
	return d, fake, paths
}

// --- inventory --------------------------------------------------------

func TestParseInventory(t *testing.T) {
	entries, err := parseInventory(strings.NewReader(readTestdata(t, "inventory.txt")))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Name != "demo-one" {
		t.Errorf("entry 0 name = %q", entries[0].Name)
	}
	if entries[0].UUID != "564d1156-b039-faa4-7688-dee27445cccd" {
		t.Errorf("normalized uuid = %q", entries[0].UUID)
	}
	if !strings.HasSuffix(entries[1].Path, "demo-two.vmx") {
		t.Errorf("entry 1 path = %q", entries[1].Path)
	}
}

func TestParseInventoryMissingFile(t *testing.T) {
	entries, err := readInventory(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil || entries != nil {
		t.Fatalf("missing file should be tolerated, got %v, %v", entries, err)
	}
}

func TestNormalizeUUID(t *testing.T) {
	cases := map[string]string{
		"56 4d 11 56 b0 39 fa a4-76 88 de e2 74 45 cc cd": "564d1156-b039-faa4-7688-dee27445cccd",
		"564D1156B039FAA47688DEE27445CCCD":                "564d1156-b039-faa4-7688-dee27445cccd",
		"not-a-uuid":                                      "not-a-uuid",
	}
	for in, want := range cases {
		if got := NormalizeUUID(in); got != want {
			t.Errorf("NormalizeUUID(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- vmx --------------------------------------------------------------

func TestParseVMX(t *testing.T) {
	cfg := parseVMX(strings.NewReader(readTestdata(t, "demo.vmx")))
	if cfg.DisplayName != "demo-one" {
		t.Errorf("DisplayName = %q", cfg.DisplayName)
	}
	if cfg.MemSizeMB != 4096 || cfg.NumCPUs != 2 {
		t.Errorf("mem/cpu = %d/%d", cfg.MemSizeMB, cfg.NumCPUs)
	}
	if cfg.GuestOS != "arm-ubuntu-64" {
		t.Errorf("GuestOS = %q", cfg.GuestOS)
	}
	if cfg.UUID != "564d1156-b039-faa4-7688-dee27445cccd" {
		t.Errorf("UUID = %q", cfg.UUID)
	}
}

func TestParseVMXUTF16(t *testing.T) {
	plain := "displayName = \"utf16-vm\"\nmemsize = \"2048\"\n"
	utf16 := make([]byte, 0, len(plain)*2)
	for _, r := range []byte(plain) {
		utf16 = append(utf16, r, 0)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "x.vmx")
	if err := os.WriteFile(p, utf16, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := readVMX(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DisplayName != "utf16-vm" || cfg.MemSizeMB != 2048 {
		t.Errorf("got %+v", cfg)
	}
}

// --- list / state -----------------------------------------------------

func TestListRunningParsesFixture(t *testing.T) {
	fake := &fakeExec{
		respond: func(args []string) (string, error) {
			return readTestdata(t, "vmrun_list.txt"), nil
		},
	}
	d := &Driver{runner: &runner{bin: "vmrun", hostType: "fusion", execFn: fake.exec}}
	running, err := d.listRunning(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !running["/Users/test/VM/demo-one.vmwarevm/demo-one.vmx"] {
		t.Errorf("running set = %v", running)
	}
}

func TestList(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	vms, err := d.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(vms) != 2 {
		t.Fatalf("want 2 vms, got %d", len(vms))
	}
	byName := map[string]model.VM{}
	for _, vm := range vms {
		byName[vm.Name] = vm
	}

	one := byName["demo-one"]
	if one.State != model.StateOn {
		t.Errorf("demo-one state = %s, want on", one.State)
	}
	if one.CPUs != 2 || one.MemoryMB != 4096 {
		t.Errorf("demo-one cpu/mem = %d/%d", one.CPUs, one.MemoryMB)
	}
	if one.Path != paths["demo-one"] {
		t.Errorf("demo-one path = %q", one.Path)
	}

	two := byName["demo-two"]
	if two.State != model.StateOff {
		t.Errorf("demo-two state = %s, want off", two.State)
	}
	if two.ID != "564def6c-baf4-4cd9-7d9f-4a1f2a6787c8" {
		t.Errorf("demo-two id = %q", two.ID)
	}

	if len(fake.calls) == 0 || fake.lastArgs()[0] != "list" {
		t.Errorf("expected a vmrun list call, got %v", fake.calls)
	}
}

func TestStateSuspended(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	// demo-two suspended: vmss file next to the vmx
	vmss := strings.TrimSuffix(paths["demo-two"], ".vmx") + ".vmss"
	if err := os.WriteFile(vmss, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	vms, err := d.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_ = fake
	for _, vm := range vms {
		if vm.Name == "demo-two" && vm.State != model.StateSuspended {
			t.Errorf("demo-two state = %s, want suspended", vm.State)
		}
	}
}

// --- resolve ----------------------------------------------------------

func TestResolve(t *testing.T) {
	d, _, paths := newTestEnv(t)
	ctx := context.Background()
	_ = ctx

	cases := []struct {
		ref  string
		want string
	}{
		{"demo-one", paths["demo-one"]},
		{"DEMO-ONE", paths["demo-one"]},                             // case-insensitive
		{"564d1156-b039-faa4-7688-dee27445cccd", paths["demo-one"]}, // full uuid
		{"564d1156", paths["demo-one"]},                             // bare hex prefix
		{"demo-t", paths["demo-two"]},                               // unique substring
		{paths["demo-two"], paths["demo-two"]},                      // direct path
	}
	for _, c := range cases {
		got, err := d.resolve(c.ref)
		if err != nil {
			t.Errorf("resolve(%q) error: %v", c.ref, err)
			continue
		}
		if got != c.want {
			t.Errorf("resolve(%q) = %q, want %q", c.ref, got, c.want)
		}
	}
}

func TestResolveErrors(t *testing.T) {
	d, _, _ := newTestEnv(t)

	if _, err := d.resolve("no-such-vm"); !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if _, err := d.resolve("demo"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("want ambiguous error for substring demo, got %v", err)
	}
}

// --- power ops --------------------------------------------------------

func TestPowerOps(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	ctx := context.Background()
	p := paths["demo-one"]

	tests := []struct {
		name string
		run  func() error
		want []string
	}{
		{"start", func() error { return d.Start(ctx, "demo-one", driver.StartOptions{}) },
			[]string{"start", p, "gui"}},
		{"start nogui", func() error { return d.Start(ctx, "demo-one", driver.StartOptions{NoGUI: true}) },
			[]string{"start", p, "nogui"}},
		{"stop soft", func() error { return d.Stop(ctx, "demo-one", driver.StopOptions{}) },
			[]string{"stop", p, "soft"}},
		{"stop hard", func() error { return d.Stop(ctx, "demo-one", driver.StopOptions{Hard: true}) },
			[]string{"stop", p, "hard"}},
		{"suspend", func() error { return d.Suspend(ctx, "demo-one", driver.SuspendOptions{}) },
			[]string{"suspend", p, "soft"}},
		{"pause", func() error { return d.Pause(ctx, "demo-one") },
			[]string{"pause", p}},
		{"reset", func() error { return d.Reset(ctx, "demo-one", driver.ResetOptions{Hard: true}) },
			[]string{"reset", p, "hard"}},
	}
	for _, tt := range tests {
		if err := tt.run(); err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		got := fake.waitArgsLike(tt.want)
		if strings.Join(got, " ") != strings.Join(tt.want, " ") {
			t.Errorf("%s: got args %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestResumeSuspended(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	ctx := context.Background()

	vmss := strings.TrimSuffix(paths["demo-two"], ".vmx") + ".vmss"
	if err := os.WriteFile(vmss, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// suspended VM is not in the running list; return the fixture list.
	if err := d.Resume(ctx, "demo-two"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(fake.lastArgs(), " ") != "start "+paths["demo-two"] {
		t.Errorf("resume of suspended VM: got %q", fake.lastArgs())
	}
}

func TestResumePoweredOff(t *testing.T) {
	d, _, _ := newTestEnv(t)
	if err := d.Resume(context.Background(), "demo-two"); err == nil {
		t.Error("resume of powered-off VM should fail")
	}
}

// --- runner error handling -------------------------------------------

func TestRunDetectsZeroExitError(t *testing.T) {
	r := &runner{
		bin:      "vmrun",
		hostType: "fusion",
		execFn: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("Error: Cannot open VM"), nil // exit 0, but error text
		},
	}
	if _, err := r.run(context.Background(), "stop", "/x.vmx"); err == nil {
		t.Fatal("expected error for vmrun exit-0 error output")
	} else if !errors.Is(err, driver.ErrNotFound) {
		t.Errorf("want ErrNotFound classification, got %v", err)
	}
}

func TestRunPassesHostType(t *testing.T) {
	var got []string
	r := &runner{
		bin:      "vmrun",
		hostType: "ws",
		execFn: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			got = args
			return nil, nil
		},
	}
	if _, err := r.run(context.Background(), "list"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "-T ws list" {
		t.Errorf("args = %q", got)
	}
}

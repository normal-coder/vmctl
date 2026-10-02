package vmrun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// --- clone ------------------------------------------------------------

func TestClone(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	ctx := context.Background()

	dest, err := d.Clone(ctx, "demo-two", driver.CloneOptions{Name: "demo-clone"})
	if err != nil {
		t.Fatal(err)
	}
	wantDest := filepath.Join(filepath.Dir(filepath.Dir(paths["demo-two"])),
		"demo-clone.vmwarevm", "demo-clone.vmx")
	if dest != wantDest {
		t.Errorf("dest = %q, want %q", dest, wantDest)
	}
	if got := strings.Join(fake.lastArgs(), " "); got !=
		"clone "+paths["demo-two"]+" "+wantDest+" linked -cloneName=demo-clone" {
		t.Errorf("args = %q", got)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("destination vmx not created: %v", err)
	}
	cfg, err := readVMX(dest)
	if err != nil || cfg.DisplayName != "demo-clone" {
		t.Errorf("dest displayName = %+v (%v)", cfg, err)
	}
	// The clone must be visible in the inventory.
	entries, _ := d.inventory()
	found := false
	for _, e := range entries {
		if e.Name == "demo-clone" && e.Path == dest {
			found = true
		}
	}
	if !found {
		t.Errorf("inventory missing clone entry: %+v", entries)
	}
}

func TestCloneFullAndSnapshot(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	if _, err := d.Clone(context.Background(), "demo-two", driver.CloneOptions{
		Name: "full-copy", Full: true, Snapshot: "base",
	}); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(fake.lastArgs(), " ")
	if !strings.Contains(got, " full ") || !strings.Contains(got, "-snapshot=base") {
		t.Errorf("args = %q", got)
	}
}

func TestCloneRefusedWhileRunning(t *testing.T) {
	d, _, _ := newTestEnv(t)
	_, err := d.Clone(context.Background(), "demo-one", driver.CloneOptions{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.vmrun.runningFor", "clone")) {
		t.Errorf("want powered-off error, got %v", err)
	}
}

func TestCloneNameConflict(t *testing.T) {
	d, _, _ := newTestEnv(t)
	_, err := d.Clone(context.Background(), "demo-two", driver.CloneOptions{Name: "demo-one"})
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.name.exists", "demo-one")) {
		t.Errorf("want name-conflict error, got %v", err)
	}
}

func TestCloneDestExists(t *testing.T) {
	d, _, paths := newTestEnv(t)
	parent := filepath.Dir(filepath.Dir(paths["demo-two"]))
	dest := filepath.Join(parent, "taken.vmwarevm", "taken.vmx")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := d.Clone(context.Background(), "demo-two", driver.CloneOptions{Name: "taken", Path: dest})
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.dest.exists", dest)) {
		t.Errorf("want dest-exists error, got %v", err)
	}
}

// --- create -----------------------------------------------------------

func TestCreateRequiresFrom(t *testing.T) {
	d, _, _ := newTestEnv(t)
	_, err := d.Create(context.Background(), driver.CreateOptions{Name: "bare"})
	if !errors.Is(err, driver.ErrNotSupported) {
		t.Errorf("want ErrNotSupported, got %v", err)
	}
}

func TestCreateWithConfig(t *testing.T) {
	d, _, _ := newTestEnv(t)
	dest, err := d.Create(context.Background(), driver.CreateOptions{
		Name:     "made-vm",
		From:     "demo-two",
		MemoryMB: 2048,
		CPUs:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := readVMX(dest)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MemSizeMB != 2048 || cfg.NumCPUs != 1 {
		t.Errorf("post-clone config = mem %d, cpus %d", cfg.MemSizeMB, cfg.NumCPUs)
	}
}

// --- set --------------------------------------------------------------

func TestSet(t *testing.T) {
	d, _, paths := newTestEnv(t)
	mem, cpus := 8192, 4
	name := "demo-two-renamed"
	if err := d.Set(context.Background(), "demo-two", driver.SetOptions{
		MemoryMB: &mem, CPUs: &cpus, Name: &name,
	}); err != nil {
		t.Fatal(err)
	}
	cfg, err := readVMX(paths["demo-two"])
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MemSizeMB != 8192 || cfg.NumCPUs != 4 || cfg.DisplayName != "demo-two-renamed" {
		t.Errorf("vmx after set = %+v", cfg)
	}
	// inventory DisplayName updated too
	raw, _ := os.ReadFile(d.invPath)
	if !strings.Contains(string(raw), `DisplayName = "demo-two-renamed"`) {
		t.Errorf("inventory not renamed:\n%s", raw)
	}
}

func TestSetRefusedWhileRunning(t *testing.T) {
	d, _, _ := newTestEnv(t)
	mem := 4096
	err := d.Set(context.Background(), "demo-one", driver.SetOptions{MemoryMB: &mem})
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.vmrun.runningFor", "set")) {
		t.Errorf("want powered-off error, got %v", err)
	}
}

func TestSetRenameConflict(t *testing.T) {
	d, _, _ := newTestEnv(t)
	name := "demo-one"
	err := d.Set(context.Background(), "demo-two", driver.SetOptions{Name: &name})
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.name.exists", "demo-one")) {
		t.Errorf("want name-conflict error, got %v", err)
	}
}

func TestSetEmptyRejected(t *testing.T) {
	d, _, _ := newTestEnv(t)
	if err := d.Set(context.Background(), "demo-two", driver.SetOptions{}); err == nil {
		t.Error("empty set should be rejected")
	}
}

// --- delete -----------------------------------------------------------

func TestDelete(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	if err := d.Delete(context.Background(), "demo-two"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(fake.lastArgs(), " ") != "deleteVM "+paths["demo-two"] {
		t.Errorf("args = %q", fake.lastArgs())
	}
	// bundle removed (fake emulates vmrun), inventory entry gone
	if _, err := os.Stat(filepath.Dir(paths["demo-two"])); !os.IsNotExist(err) {
		t.Errorf("bundle still exists: %v", err)
	}
	raw, _ := os.ReadFile(d.invPath)
	if strings.Contains(string(raw), "demo-two") {
		t.Errorf("inventory still mentions demo-two:\n%s", raw)
	}
}

func TestDeleteRefusedWhileRunning(t *testing.T) {
	d, _, _ := newTestEnv(t)
	err := d.Delete(context.Background(), "demo-one")
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.vmrun.runningFor", "delete")) {
		t.Errorf("want powered-off error, got %v", err)
	}
}

// --- inventory mutation -----------------------------------------------

func TestRegisterAndUnregisterVM(t *testing.T) {
	dir := t.TempDir()
	inv := filepath.Join(dir, "vmInventory")
	vmx := filepath.Join(dir, "new.vmwarevm", "new.vmx")

	// registerVM only appends to an existing inventory (it never
	// creates Fusion's files from scratch).
	if err := os.WriteFile(inv, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := registerVM(inv, vmx, "first", "56 4d aa bb cc dd ee ff-01 02 03 04 05 06 07 08"); err != nil {
		t.Fatal(err)
	}
	entries, err := readInventory(inv)
	if err != nil || len(entries) != 1 || entries[0].Name != "first" {
		t.Fatalf("after register: %+v, %v", entries, err)
	}
	if entries[0].UUID != "564daabb-ccdd-eeff-0102-030405060708" {
		t.Errorf("registered uuid = %q", entries[0].UUID)
	}
	// re-register updates instead of duplicating
	if err := registerVM(inv, vmx, "renamed", ""); err != nil {
		t.Fatal(err)
	}
	entries, _ = readInventory(inv)
	if len(entries) != 1 || entries[0].Name != "renamed" {
		t.Fatalf("after update: %+v", entries)
	}
	if entries[0].UUID != "564daabb-ccdd-eeff-0102-030405060708" {
		t.Errorf("empty uuid should keep existing, got %q", entries[0].UUID)
	}
	if err := unregisterVM(inv, vmx); err != nil {
		t.Fatal(err)
	}
	entries, _ = readInventory(inv)
	if len(entries) != 0 {
		t.Fatalf("after unregister: %+v", entries)
	}
}

func TestRegisterSkipsMissingInventory(t *testing.T) {
	if err := registerVM(filepath.Join(t.TempDir(), "nope"), "/x/y.vmx", "n", ""); err != nil {
		t.Errorf("missing inventory should not error: %v", err)
	}
}

// --- uuid fill --------------------------------------------------------

// vmrun blanks uuid.bios when cloning; fillCloneUUID must give the
// new vmx an identity immediately, and leave existing ones alone.
func TestFillCloneUUID(t *testing.T) {
	dir := t.TempDir()
	vmx := filepath.Join(dir, "clone.vmx")

	// empty uuid.bios -> generated
	if err := os.WriteFile(vmx, []byte("displayName = \"c\"\nuuid.bios = \"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := fillCloneUUID(vmx)
	if got == "" {
		t.Fatal("expected a generated uuid")
	}
	cfg, err := readVMX(vmx)
	if err != nil {
		t.Fatal(err)
	}
	if NormalizeUUID(got) != cfg.UUID {
		t.Errorf("vmx uuid %q != inventory uuid %q", cfg.UUID, got)
	}
	if !strings.HasPrefix(got, "56 4d ") {
		t.Errorf("uuid %q missing VMware prefix", got)
	}

	// existing uuid kept as-is
	if err := os.WriteFile(vmx, []byte("uuid.bios = \"56 4d 11 56 b0 39 fa a4-76 88 de e2 74 45 cc cd\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := fillCloneUUID(vmx); NormalizeUUID(got) != "564d1156-b039-faa4-7688-dee27445cccd" {
		t.Errorf("existing uuid overwritten: %q", got)
	}
}

// --- name validation --------------------------------------------------

func TestValidateName(t *testing.T) {
	bad := []string{"", "  ", "a/b", `a\b`, `a"b`, "a\nb"}
	for _, n := range bad {
		if err := validateName(n); err == nil {
			t.Errorf("validateName(%q) should fail", n)
		}
	}
	for _, n := range []string{"my-vm", "ubuntu server 24.04", "测试虚拟机"} {
		if err := validateName(n); err != nil {
			t.Errorf("validateName(%q) = %v", n, err)
		}
	}
}

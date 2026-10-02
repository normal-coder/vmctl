package vmrun

import (
	"context"
	"os"
	"strings"
	"testing"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

func TestParseSnapshotList(t *testing.T) {
	flat := "Total snapshots: 2\nsnap1\nsnap2\n"
	got := parseSnapshotList(flat)
	if len(got) != 2 || got[0].Name != "snap1" || got[1].Name != "snap2" {
		t.Fatalf("flat parse: %+v", got)
	}
	if got[0].Depth != 0 || got[1].Depth != 0 {
		t.Errorf("flat depths should be 0: %+v", got)
	}

	tree := "Total snapshots: 3\nsnap1\n\tsnap2\n\t\tsnap3\n"
	got = parseSnapshotList(tree)
	if len(got) != 3 {
		t.Fatalf("tree parse: %+v", got)
	}
	if got[1].Depth != 1 || got[2].Depth != 2 {
		t.Errorf("tree depths: %+v", got)
	}
	if got[2].Name != "snap3" {
		t.Errorf("tree name: %+v", got[2])
	}

	if got := parseSnapshotList("Total snapshots: 0\n"); len(got) != 0 {
		t.Errorf("empty list: %+v", got)
	}
}

func TestSnapshotCreateDeleteRevert(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	ctx := context.Background()

	if err := d.SnapshotCreate(ctx, "demo-two", "snap-a"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.lastArgs(), " "); got != "snapshot "+paths["demo-two"]+" snap-a" {
		t.Errorf("create args = %q", got)
	}

	if err := d.SnapshotDelete(ctx, "demo-two", "snap-a", true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.lastArgs(), " "); got !=
		"deleteSnapshot "+paths["demo-two"]+" snap-a andDeleteChildren" {
		t.Errorf("delete args = %q", got)
	}
	if err := d.SnapshotDelete(ctx, "demo-two", "", false); err == nil {
		t.Error("empty snapshot name should be rejected")
	}

	if err := d.SnapshotRevert(ctx, "demo-two", "snap-a"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.lastArgs(), " "); got != "revertToSnapshot "+paths["demo-two"]+" snap-a" {
		t.Errorf("revert args = %q", got)
	}
}

// vmrun sometimes exits 0 while printing "Error: ..." — the message
// must reach the caller intact.
func TestSnapshotCreateSurfacesExitZeroError(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		return "Error: A snapshot with the name already exists", nil
	}
	err := d.SnapshotCreate(context.Background(), "demo-two", "snap-a")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate create: %v", err)
	}
}

func TestSnapshotsListing(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		return "Total snapshots: 2\nbase\n\tderived\n", nil
	}
	snaps, err := d.Snapshots(context.Background(), "demo-two", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 || snaps[1].Name != "derived" || snaps[1].Depth != 1 {
		t.Errorf("snaps: %+v", snaps)
	}
	if !strings.Contains(strings.Join(fake.lastArgs(), " "), "showTree") {
		t.Errorf("tree flag not passed: %v", fake.lastArgs())
	}
}

func TestSnapshotsResolveError(t *testing.T) {
	d, _, _ := newTestEnv(t)
	if _, err := d.Snapshots(context.Background(), "nope", false); err == nil {
		t.Error("unknown ref should error")
	}
}

// --- exec -------------------------------------------------------------

// stripAuth removes leading -gu/-gp pairs after the -T prefix.
func stripAuth(args []string) []string {
	args = stripHostType(args)
	for len(args) >= 2 && (args[0] == "-gu" || args[0] == "-gp") {
		args = args[2:]
	}
	return args
}

func execOpts(script, user, pass string) driver.ExecOptions {
	return driver.ExecOptions{Script: script, User: user, Password: pass}
}

func TestExecCapturesOutputAndExitCode(t *testing.T) {
	d, fake, paths := newTestEnv(t)
	// demo-one is "running" in the fake's list output.
	var script string
	var deleted []string
	fake.respond = func(args []string) (string, error) {
		cmd := stripAuth(args)
		switch cmd[0] {
		case "list":
			return "Total running VMs: 1\n" + paths["demo-one"] + "\n", nil
		case "runScriptInGuest":
			script = cmd[3]
			return "", nil
		case "CopyFileFromGuestToHost":
			guest, host := cmd[2], cmd[3]
			content := "hello from guest\n"
			if strings.HasSuffix(guest, ".code") {
				content = "7\n"
			}
			return "", os.WriteFile(host, []byte(content), 0o644)
		case "deleteFileInGuest":
			deleted = append(deleted, cmd[2])
			return "", nil
		}
		return "", nil
	}

	out, code, err := d.Exec(context.Background(), "demo-one", execOpts("echo hi", "bob", "s3cret"))
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello from guest\n" || code != 7 {
		t.Errorf("out=%q code=%d", out, code)
	}
	if !strings.Contains(script, `sh -c 'echo hi'`) {
		t.Errorf("script not wrapped: %q", script)
	}
	if !strings.Contains(script, ">'/tmp/.vmctl-exec-") {
		t.Errorf("script missing capture redirect: %q", script)
	}

	// Auth flags must precede the command: -T fusion -gu bob -gp s3cret runScriptInGuest ...
	var execCall []string
	for _, c := range fake.calls {
		if len(c) > 6 && c[6] == "runScriptInGuest" {
			execCall = c
		}
	}
	if execCall == nil {
		t.Fatalf("runScriptInGuest never called: %v", fake.calls)
	}
	if execCall[2] != "-gu" || execCall[3] != "bob" || execCall[4] != "-gp" || execCall[5] != "s3cret" {
		t.Errorf("auth placement: %v", execCall)
	}
	if len(deleted) != 2 {
		t.Errorf("guest temp files not cleaned: %v", deleted)
	}
}

func TestExecRefusedWhenPoweredOff(t *testing.T) {
	d, _, _ := newTestEnv(t)
	_, _, err := d.Exec(context.Background(), "demo-two", execOpts("true", "", ""))
	if err == nil || !strings.Contains(err.Error(), i18n.T("err.vmrun.notRunningFor", "exec")) {
		t.Errorf("want powered-on error, got %v", err)
	}
}

func TestExecEmptyScript(t *testing.T) {
	d, _, _ := newTestEnv(t)
	if _, _, err := d.Exec(context.Background(), "demo-one", execOpts("  ", "", "")); err == nil {
		t.Error("empty script should be rejected")
	}
}

// --- guest ip ---------------------------------------------------------

func TestGuestIP(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) {
		return "Guest IP: 192.168.38.77", nil
	}

	ip, err := d.GuestIP(context.Background(), "demo-one", false)
	if err != nil || ip != "192.168.38.77" {
		t.Fatalf("ip=%q err=%v", ip, err)
	}
	if got := strings.Join(fake.lastArgs(), " "); strings.Contains(got, "-wait") {
		t.Errorf("unexpected -wait: %q", got)
	}

	if _, err := d.GuestIP(context.Background(), "demo-one", true); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(fake.lastArgs(), " "); !strings.HasSuffix(got, "-wait") {
		t.Errorf("-wait not passed: %q", got)
	}
}

func TestGuestIPEmptyOutput(t *testing.T) {
	d, fake, _ := newTestEnv(t)
	fake.respond = func(args []string) (string, error) { return "", nil }
	if _, err := d.GuestIP(context.Background(), "demo-one", false); err == nil {
		t.Error("empty IP should error")
	}
}

// --- helpers ----------------------------------------------------------

func TestShQuote(t *testing.T) {
	cases := map[string]string{
		"simple":     "'simple'",
		"a b":        "'a b'",
		"it's":       `'it'\''s'`,
		"a;rm -rf /": `'a;rm -rf /'`,
	}
	for in, want := range cases {
		if got := shQuote(in); got != want {
			t.Errorf("shQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestGuestAuthArgs(t *testing.T) {
	if got := guestAuthArgs(execOpts("", "", "")); got != nil {
		t.Errorf("no user: %v", got)
	}
	if got := guestAuthArgs(execOpts("", "bob", "")); strings.Join(got, " ") != "-gu bob" {
		t.Errorf("user only: %v", got)
	}
	if got := guestAuthArgs(execOpts("", "bob", "pw")); strings.Join(got, " ") != "-gu bob -gp pw" {
		t.Errorf("user+pass: %v", got)
	}
}

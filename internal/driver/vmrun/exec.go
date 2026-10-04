package vmrun

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/driver"
	"gitee.com/normalcoder/vmctl/internal/i18n"
)

// Exec implements driver.Driver.
//
// vmrun cannot stream a guest program's output to the host, so the
// command runs as `sh -c '<script>' > /tmp/...  2>&1` inside the
// guest, its exit code is written next to it, and both files are
// copied back. Wrapping the user script as a single -c argument keeps
// syntax errors inside the inner shell: they land in the captured
// output with a non-zero exit code instead of breaking the wrapper.
func (d *Driver) Exec(ctx context.Context, ref string, opts driver.ExecOptions) (string, int, error) {
	if strings.TrimSpace(opts.Script) == "" {
		return "", 0, errors.New(i18n.T("err.exec.empty"))
	}
	path, err := d.resolve(ref)
	if err != nil {
		return "", 0, err
	}
	if err := d.ensureRunning(ctx, path, "exec"); err != nil {
		return "", 0, err
	}
	auth := guestAuthArgs(opts)

	tag, err := randomTag()
	if err != nil {
		return "", 0, err
	}
	outGuest := "/tmp/.vmctl-exec-" + tag
	codeGuest := outGuest + ".code"

	script := fmt.Sprintf("sh -c %s >%s 2>&1; echo $? >%s",
		shQuote(opts.Script), shQuote(outGuest), shQuote(codeGuest))
	if _, err := d.runner.run(ctx, withAuth(auth, "runScriptInGuest", path, "/bin/sh", script)...); err != nil {
		return "", 0, err
	}

	// Fetch results back; guest-side temp files are removed best effort.
	hostOut, err := os.CreateTemp("", "vmctl-exec-out-*")
	if err != nil {
		return "", 0, err
	}
	hostCode, err := os.CreateTemp("", "vmctl-exec-code-*")
	if err != nil {
		os.Remove(hostOut.Name())
		return "", 0, err
	}
	defer func() {
		os.Remove(hostOut.Name())
		os.Remove(hostCode.Name())
		d.deleteGuestFile(ctx, auth, path, outGuest)
		d.deleteGuestFile(ctx, auth, path, codeGuest)
	}()

	if _, err := d.runner.run(ctx, withAuth(auth,
		"CopyFileFromGuestToHost", path, outGuest, hostOut.Name())...); err != nil {
		return "", 0, fmt.Errorf(i18n.T("err.exec.fetchOutput"), err)
	}
	if _, err := d.runner.run(ctx, withAuth(auth,
		"CopyFileFromGuestToHost", path, codeGuest, hostCode.Name())...); err != nil {
		return "", 0, fmt.Errorf(i18n.T("err.exec.fetchExit"), err)
	}

	code := -1
	if raw, err := os.ReadFile(hostCode.Name()); err == nil {
		if c, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			code = c
		}
	}
	out, err := os.ReadFile(hostOut.Name())
	if err != nil {
		return "", code, err
	}
	return string(out), code, nil
}

// GuestIP implements driver.Driver.
func (d *Driver) GuestIP(ctx context.Context, ref string, wait bool) (string, error) {
	path, err := d.resolve(ref)
	if err != nil {
		return "", err
	}
	args := []string{"getGuestIPAddress", path}
	if wait {
		args = append(args, "-wait")
	}
	out, err := d.runner.run(ctx, args...)
	if err != nil {
		return "", err
	}
	ip := parseGuestIP(out)
	if ip == "" {
		return "", errors.New(i18n.T("err.vsphere.noIP"))
	}
	return ip, nil
}

// deleteGuestFile removes a temp file in the guest, ignoring failures.
func (d *Driver) deleteGuestFile(ctx context.Context, auth []string, vmxPath, guestPath string) {
	_, _ = d.runner.run(ctx, withAuth(auth, "deleteFileInGuest", vmxPath, guestPath)...)
}

// guestAuthArgs maps ExecOptions onto vmrun's global -gu/-gp flags.
// Passwords are only passed when a user is set — an empty -gu with a
// -gp is meaningless to vmrun.
func guestAuthArgs(opts driver.ExecOptions) []string {
	if opts.User == "" {
		return nil
	}
	a := []string{"-gu", opts.User}
	if opts.Password != "" {
		a = append(a, "-gp", opts.Password)
	}
	return a
}

// withAuth builds one vmrun invocation: global auth flags first,
// then the command and its parameters.
func withAuth(auth []string, cmd ...string) []string {
	args := make([]string, 0, len(auth)+len(cmd))
	args = append(args, auth...)
	return append(args, cmd...)
}

// shQuote wraps s in single quotes for /bin/sh.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// randomTag returns a hex token for guest temp file names.
func randomTag() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

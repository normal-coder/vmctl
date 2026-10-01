package vmrun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"gitee.com/normalcoder/vmctl/internal/driver"
)

// execFunc runs a command and returns combined output on failure.
// It exists as an injection point for tests.
type execFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultExec runs a real subprocess, merging stderr into the output on error.
func defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		combined := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		return []byte(strings.TrimSpace(combined)), err
	}
	return stdout.Bytes(), nil
}

// runner wraps a vmrun binary invocation.
type runner struct {
	bin      string
	hostType string
	execFn   execFunc
}

// Error is a failed vmrun invocation.
type Error struct {
	Args   []string
	Output string // vmrun stdout/stderr
	Err    error  // underlying exec error, nil if vmrun exited 0
}

func (e *Error) Error() string {
	msg := strings.TrimSpace(e.Output)
	if msg == "" {
		if e.Err != nil {
			msg = e.Err.Error()
		} else {
			msg = "vmrun failed"
		}
	}
	return msg
}

func (e *Error) Unwrap() error { return e.Err }

// run executes vmrun with the authentication/host-type prefix applied.
// Some vmrun versions exit 0 while printing "Error: ..." — those are
// normalized into real errors here.
func (r *runner) run(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"-T", r.hostType}, args...)
	out, err := r.execFn(ctx, r.bin, full...)
	text := strings.TrimSpace(string(out))

	if err == nil && vmrunReportedError(text) {
		err = errors.New("vmrun reported an error with exit code 0")
	}
	if err != nil {
		return "", classify(&Error{Args: args, Output: text, Err: err})
	}
	return text, nil
}

// vmrunReportedError detects success-exit-with-error-output quirks.
func vmrunReportedError(text string) bool {
	return strings.HasPrefix(text, "Error:")
}

// classify maps recognizable vmrun failures onto sentinel errors.
func classify(err error) error {
	var ve *Error
	if !errors.As(err, &ve) {
		return err
	}
	msg := strings.ToLower(ve.Error())
	switch {
	case strings.Contains(msg, "cannot open vm"),
		strings.Contains(msg, "could not find"),
		strings.Contains(msg, "not found"),
		strings.Contains(msg, "no such"):
		return fmt.Errorf("%w: %s", driver.ErrNotFound, ve.Error())
	}
	return err
}

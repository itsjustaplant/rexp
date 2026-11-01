package programstatus

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestWireFormatAndStates(t *testing.T) {
	var output bytes.Buffer
	reporter := New(&output)
	ctx := WithReporter(context.Background(), reporter)
	Working(ctx, "Starting ✓")
	Approval(ctx, "Delete project?")
	Working(ctx, "Deleting")
	reporter.Finish(nil, "rexp delete")
	want := ""
	for _, report := range []struct{ state, kind, message string }{
		{"working", "", "Starting ✓"}, {"blocked", "permission", "Delete project?"},
		{"working", "", "Deleting"}, {"done", "", "rexp delete completed"},
	} {
		want += "\x1b]7501;state=" + report.state + ":id=rexp:app=rexp"
		if report.kind != "" {
			want += ":kind=" + report.kind
		}
		want += ":msg=" + base64.StdEncoding.EncodeToString([]byte(report.message)) + "\x1b\\"
	}
	if output.String() != want {
		t.Fatalf("wire format: %q, want %q", output.String(), want)
	}
}

func TestFinishErrorCancellationAndHelp(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		idle bool
		want string
	}{
		{"help", nil, false, ""},
		{"error", errors.New("failure"), false, "state=error"},
		{"interrupt", context.Canceled, false, "state=idle"},
		{"deadline", context.DeadlineExceeded, false, "state=error"},
		{"declined", nil, true, "state=idle"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			reporter := New(&output)
			if test.idle {
				Idle(WithReporter(context.Background(), reporter), "Cancelled")
			}
			reporter.Finish(test.err, "rexp")
			if test.want == "" {
				if output.Len() != 0 {
					t.Fatalf("help emitted status: %q", output.String())
				}
				return
			}
			if !strings.Contains(output.String(), test.want) || strings.Contains(output.String(), "state=done") {
				t.Fatalf("status: %q", output.String())
			}
			if test.idle && strings.Count(output.String(), "\x1b]7501;") != 1 {
				t.Fatal("idle was overwritten at finish")
			}
		})
	}
}

func TestMessageSanitizationAndLimits(t *testing.T) {
	message := "\xffhello\n\t\x1b\u009f" + strings.Repeat("✓", 2000)
	var output bytes.Buffer
	Working(WithReporter(context.Background(), New(&output)), message)
	sequence := output.String()
	if len(sequence) > 4096 {
		t.Fatalf("sequence too long: %d", len(sequence))
	}
	encoded := strings.TrimSuffix(strings.Split(sequence, ":msg=")[1], "\x1b\\")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) > 2048 || !utf8.Valid(decoded) {
		t.Fatalf("invalid decoded message: %d %v", len(decoded), err)
	}
	for _, r := range string(decoded) {
		if unicode.IsControl(r) {
			t.Fatalf("control character remained: %U", r)
		}
	}
}

func TestRedirectedStderrReceivesNoSequences(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reporter := NewTerminal(file)
	Working(WithReporter(context.Background(), reporter), "Running")
	reporter.Finish(nil, "rexp")
	info, err := file.Stat()
	if err != nil || info.Size() != 0 {
		t.Fatalf("redirected stderr contains status: %v", err)
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	if NewTerminal(devNull).writer != nil {
		t.Fatal("character device /dev/null was mistaken for a terminal")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestMissingOrFailedReporterIsNonfatal(t *testing.T) {
	Working(context.Background(), "No reporter")
	Working(nil, "No context")
	var reporter *Reporter
	reporter.Finish(errors.New("failed"), "rexp")
	Working(WithReporter(context.Background(), New(failedWriter{})), "Closed terminal")
}

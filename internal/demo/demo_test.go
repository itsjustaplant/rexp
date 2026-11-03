package demo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestDemoRendersLogoAndStopsClockCleanly(t *testing.T) {
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"logo"}, &out); err != nil || !strings.Contains(out.String(), "rexp") {
		t.Fatalf("logo: %q %v", out.String(), err)
	}
	out.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, []string{"clock"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Local time") || !strings.HasSuffix(out.String(), "\x1b[?25h\n") {
		t.Fatal("clock did not render or restore its cursor")
	}
	if err := Run(context.Background(), []string{"clock"}, brokenWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("ignored output failure: %v", err)
	}
	if err := Run(context.Background(), []string{"unknown"}, &out); err == nil {
		t.Fatal("accepted unknown demo pane")
	}
}

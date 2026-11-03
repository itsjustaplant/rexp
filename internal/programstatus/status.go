// Package programstatus emits rexp's OSC 7501 lifecycle reports. It is an
// application integration, not a general implementation of every protocol key.
package programstatus

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"unicode"
)

type State string

const (
	WorkingState State = "working"
	BlockedState State = "blocked"
	DoneState    State = "done"
	ErrorState   State = "error"
	IdleState    State = "idle"
)

// Reporter uses its own named record to avoid overwriting status reports from
// editors or other programs running in the same terminal.
type Reporter struct {
	mu      sync.Mutex
	writer  io.Writer
	state   State
	started bool
}

func New(writer io.Writer) *Reporter { return &Reporter{writer: writer} }

// IsTerminal reports whether file is an interactive terminal without reading it.
func IsTerminal(file *os.File) bool { return file != nil && isTerminal(file) }

// NewTerminal never queries stdin for protocol support. Unknown OSC sequences
// may safely be ignored; redirected stderr and dumb terminals receive nothing.
func NewTerminal(stderr *os.File) *Reporter {
	if stderr == nil || os.Getenv("TERM") == "dumb" || os.Getenv("REXP_NO_STATUS") == "1" || !isTerminal(stderr) {
		return New(nil)
	}
	return New(stderr)
}

// report sanitizes free text into a single UTF-8 line before base64 encoding,
// keeping it within the protocol's 2048 decoded-byte message limit.
func (reporter *Reporter) report(state State, kind, message string) {
	if reporter == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	reporter.reportLocked(state, kind, message)
}

func (reporter *Reporter) reportLocked(state State, kind, message string) {
	reporter.started = true
	reporter.state = state
	if reporter.writer == nil {
		return
	}
	payload := "state=" + string(state) + ":id=rexp:app=rexp"
	if state == BlockedState {
		payload += ":kind=" + kind
	}
	payload += ":msg=" + base64.StdEncoding.EncodeToString([]byte(singleLine(message)))
	// Status output is advisory: a closed terminal must not fail project work.
	_, _ = io.WriteString(reporter.writer, "\x1b]7501;"+payload+"\x1b\\")
}

func singleLine(message string) string {
	var result strings.Builder
	for _, r := range strings.ToValidUTF8(message, "�") {
		if unicode.IsControl(r) {
			r = ' '
		}
		if result.Len()+len(string(r)) > 2048 {
			break
		}
		result.WriteRune(r)
	}
	return result.String()
}

// Finish preserves idle after a declined operation, leaves help/version quiet,
// and reports argument/setup errors even if work never reached a command hook.
func (reporter *Reporter) Finish(err error, command string) {
	if reporter == nil {
		return
	}
	reporter.mu.Lock()
	defer reporter.mu.Unlock()
	switch {
	case errors.Is(err, context.Canceled):
		reporter.reportLocked(IdleState, "", command+" cancelled")
	case err != nil:
		reporter.reportLocked(ErrorState, "", command+" failed")
	case reporter.started && reporter.state != IdleState:
		reporter.reportLocked(DoneState, "", command+" completed")
	}
}

type contextKey struct{}

func WithReporter(ctx context.Context, reporter *Reporter) context.Context {
	return context.WithValue(ctx, contextKey{}, reporter)
}

func fromContext(ctx context.Context) *Reporter {
	if ctx == nil {
		return nil
	}
	reporter, _ := ctx.Value(contextKey{}).(*Reporter)
	return reporter
}

func Working(ctx context.Context, message string) { fromContext(ctx).report(WorkingState, "", message) }
func Approval(ctx context.Context, message string) {
	fromContext(ctx).report(BlockedState, "permission", message)
}
func Editing(ctx context.Context, message string) {
	fromContext(ctx).report(BlockedState, "question", message)
}
func Idle(ctx context.Context, message string) { fromContext(ctx).report(IdleState, "", message) }

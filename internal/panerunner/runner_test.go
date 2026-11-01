package panerunner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPaneHelperProcess(t *testing.T) {
	if os.Getenv("REXP_RUNNER_TEST") != "1" {
		return
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(9)
	}
	switch args[0] {
	case "record":
		file, err := os.OpenFile(args[1], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			os.Exit(8)
		}
		_ = json.NewEncoder(file).Encode(args[2:])
		_ = file.Close()
	case "fail":
		os.Exit(7)
	case "streams":
		data, _ := io.ReadAll(os.Stdin)
		fmt.Fprint(os.Stdout, string(data))
		fmt.Fprint(os.Stderr, "stderr")
	case "wait":
		if err := os.WriteFile(args[1], []byte("started"), 0o600); err != nil {
			os.Exit(8)
		}
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

func helper(t *testing.T, args ...string) []string {
	t.Helper()
	t.Setenv("REXP_RUNNER_TEST", "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return append([]string{exe, "-test.run=^TestPaneHelperProcess$", "--"}, args...)
}

func encoded(t *testing.T, commands [][]string) string {
	t.Helper()
	invocation, err := Invocation(commands)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(invocation[0]) || invocation[1] != Mode {
		t.Fatalf("invalid invocation %v", invocation)
	}
	return invocation[2]
}

func TestOrderedLiteralArgumentsAndStopOnFailure(t *testing.T) {
	log := filepath.Join(t.TempDir(), "order")
	commands := [][]string{helper(t, "record", log, "first", "a file", "$(touch unwanted)"), helper(t, "record", log, "second"), helper(t, "fail"), helper(t, "record", log, "must-not-run")}
	code, err := Run(context.Background(), []string{encoded(t, commands)}, nil, io.Discard, io.Discard)
	if code != 7 || err == nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var lines [][]string
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var args []string
		if err := json.Unmarshal(line, &args); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, args)
	}
	if !reflect.DeepEqual(lines, [][]string{{"first", "a file", "$(touch unwanted)"}, {"second"}}) {
		t.Fatalf("commands changed or ran after failure: %v", lines)
	}
}

func TestRunnerStreamsAndSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := Run(context.Background(), []string{encoded(t, [][]string{helper(t, "streams")})}, strings.NewReader("input"), &stdout, &stderr)
	if code != 0 || err != nil || stdout.String() != "input" || stderr.String() != "stderr" {
		t.Fatalf("code=%d error=%v stdout=%q stderr=%q", code, err, stdout.String(), stderr.String())
	}
}

func TestRunnerRejectsEntireInvalidSequenceBeforeStarting(t *testing.T) {
	log := filepath.Join(t.TempDir(), "must-not-exist")
	data, _ := json.Marshal(Payload{Version: 1, Commands: [][]string{helper(t, "record", log), {}}})
	code, err := Run(context.Background(), []string{base64.RawURLEncoding.EncodeToString(data)}, nil, io.Discard, io.Discard)
	if code != 2 || err == nil {
		t.Fatalf("accepted invalid sequence: %d %v", code, err)
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("started before validating entire payload")
	}
	for _, input := range []string{"not-base64!", base64.RawURLEncoding.EncodeToString([]byte(`{"version":2,"commands":[["echo"]]}`)), base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"commands":[],"unknown":true}`))} {
		if _, err := Decode(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
}

func TestRunnerCancellationStopsLaterCommands(t *testing.T) {
	ready, log := filepath.Join(t.TempDir(), "ready"), filepath.Join(t.TempDir(), "later")
	commands := [][]string{helper(t, "wait", ready), helper(t, "record", log)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan int, 1)
	payload := encoded(t, commands)
	go func() {
		code, _ := Run(ctx, []string{payload}, nil, io.Discard, io.Discard)
		finished <- code
	}()
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
wait:
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(ready); err == nil {
				break wait
			}
		case <-deadline:
			t.Fatal("child did not start")
		}
	}
	cancel()
	select {
	case code := <-finished:
		if code != 130 {
			t.Fatalf("code: %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runner ignored cancellation")
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("ran a later command after cancellation")
	}
}

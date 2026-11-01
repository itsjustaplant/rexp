package panerunner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestEffectivePathDoesNotFallBackToParent(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on parent PATH")
	}
	invocation, err := InvocationWithEnvironment([][]string{{"go", "version"}}, map[string]string{"PATH": t.TempDir()}, false)
	if err != nil {
		t.Fatal(err)
	}
	if code, err := Run(context.Background(), invocation[2:], nil, io.Discard, io.Discard); code != 127 || err == nil {
		t.Fatalf("fell back to parent PATH: code=%d err=%v", code, err)
	}
}

func TestRelativePathEntryRequiresExplicitProgramPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable permission check")
	}
	t.Chdir(t.TempDir())
	if err := os.WriteFile("probe", []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := findExecutable("probe", []string{"PATH=."}); !errors.Is(err, exec.ErrDot) {
		t.Fatalf("relative PATH accepted implicitly: %v", err)
	}
	if program, err := findExecutable("./probe", []string{"PATH=."}); err != nil || program != "./probe" {
		t.Fatalf("explicit path rejected: %q %v", program, err)
	}
}

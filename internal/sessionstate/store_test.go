package sessionstate

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T, home string) *Store {
	t.Helper()
	store, err := Open(home, Ref{Scope: Scope{HostID: "host", Account: "account"}, Workspace: "global:demo"})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestStoreRecordsAndCleanupDoNotConfuseReplacementSessions(t *testing.T) {
	home := t.TempDir()
	store := testStore(t, home)
	if record, known, err := store.Read(); err != nil || record != nil || known {
		t.Fatalf("new store: %v %v %v", record, known, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".rexp")); !os.IsNotExist(err) {
		t.Fatal("read created state")
	}
	release, err := store.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save("", "creating"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("old", "starting"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save("replacement", "ready"); err != nil {
		t.Fatal(err)
	}
	release()
	if err := store.ClearIfSession(context.Background(), "old"); err != nil {
		t.Fatal(err)
	}
	record, known, err := store.Read()
	if err != nil || record == nil || record.Session != "replacement" || !known {
		t.Fatalf("old observer lost replacement: %v %v", record, err)
	}
	if err := store.ClearIfSession(context.Background(), "replacement"); err != nil {
		t.Fatal(err)
	}
	record, known, err = store.Read()
	if err != nil || record != nil || !known {
		t.Fatal("cleanup did not retain migration history")
	}
	if _, err := os.Stat(store.Path()); !os.IsNotExist(err) {
		t.Fatal("closed-session record was not removed")
	}
}

func TestStoreRejectsCorruptionAndSymlinks(t *testing.T) {
	store := testStore(t, t.TempDir())
	release, err := store.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := store.Save("id", "ready"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Read(); err == nil {
		t.Fatal("accepted invalid record")
	}
	if err := store.Save("other", "ready"); err == nil {
		t.Fatal("overwrote corrupt state silently")
	}
	// Explicit recovery can forget corrupt JSON, without following symlinks.
	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "keep")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, store.Path()); err != nil {
		t.Skip(err)
	}
	if err := store.Clear(); err == nil {
		t.Fatal("accepted symlink record")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "keep" {
		t.Fatal("modified symlink target")
	}
}

func TestStoreLockHelperProcess(t *testing.T) {
	home := os.Getenv("REXP_TEST_LOCK_HOME")
	if home == "" {
		return
	}
	store := testStore(t, home)
	release, err := store.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	os.Stdout.WriteString("locked\n")
	time.Sleep(time.Hour)
}

func TestStoreLockSurvivesContentionAndReleasesOnProcessDeath(t *testing.T) {
	home := t.TempDir()
	child := exec.Command(os.Args[0], "-test.run=^TestStoreLockHelperProcess$")
	child.Env = append(os.Environ(), "REXP_TEST_LOCK_HOME="+home)
	pipe, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer child.Process.Kill()
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() || scanner.Text() != "locked" {
		t.Fatal("child did not acquire lock")
	}
	store := testStore(t, home)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if release, err := store.Lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		if release != nil {
			release()
		}
		t.Fatalf("contended lock ignored cancellation: %v", err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	release, err := store.Lock(ctx)
	if err != nil {
		t.Fatalf("dead process left stale lock: %v", err)
	}
	release()
}

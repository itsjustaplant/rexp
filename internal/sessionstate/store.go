// Package sessionstate keeps private workspace-to-session associations.
package sessionstate

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Scope struct {
	HostID  string `json:"host_id"`
	Account string `json:"account"`
}

type Ref struct {
	Scope     Scope  `json:"scope"`
	Workspace string `json:"workspace"`
}

type Record struct {
	Version int    `json:"version"`
	Ref     Ref    `json:"ref"`
	Session string `json:"session_id"`
	State   string `json:"state"`
}

type Store struct {
	Ref  Ref
	path string
}

func Open(home string, ref Ref) (*Store, error) {
	if !filepath.IsAbs(home) || ref.Scope.HostID == "" || ref.Scope.Account == "" || ref.Workspace == "" {
		return nil, errors.New("invalid workspace session identity")
	}
	key, _ := json.Marshal(ref)
	hash := sha256.Sum256(key)
	return &Store{Ref: ref, path: filepath.Join(home, ".rexp", "state", "sessions", fmt.Sprintf("%x.json", hash[:]))}, nil
}

func (store *Store) Path() string { return store.path }

// Lock is an OS lock released even when the process dies. Lock files stay in
// place so concurrent processes always lock the same inode.
func (store *Store) Lock(ctx context.Context) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(store.path), 0700); err != nil {
		return nil, err
	}
	path := store.path + ".lock"
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("session lock must be a regular file: %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			file.Close()
			return nil, err
		}
		ok, err := tryLock(file)
		if err != nil {
			file.Close()
			return nil, err
		}
		if ok {
			return func() { unlock(file); file.Close() }, nil
		}
		select {
		case <-ctx.Done():
			file.Close()
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func regular(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("session state must be a regular file: %s", path)
	}
	return true, nil
}

// Read never creates files. Known distinguishes a closed managed workspace
// from one eligible for first-time migration from legacy labels.
func (store *Store) Read() (record *Record, known bool, err error) {
	known, err = regular(store.path + ".known")
	if err != nil {
		return nil, false, err
	}
	exists, err := regular(store.path)
	if err != nil || !exists {
		return nil, known, err
	}
	file, err := os.Open(store.path)
	if err != nil {
		return nil, known, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	decoder.DisallowUnknownFields()
	var saved Record
	if err := decoder.Decode(&saved); err != nil {
		return nil, known, fmt.Errorf("read session record %s: %w", store.path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, known, fmt.Errorf("session record must contain one JSON object: %s", store.path)
	}
	if saved.Version != 1 || saved.Ref != store.Ref || (saved.State != "creating" && saved.State != "starting" && saved.State != "ready") || (saved.State == "creating") != (saved.Session == "") {
		return nil, known, fmt.Errorf("invalid session record: %s", store.path)
	}
	return &saved, true, nil
}

// Save and Clear require the caller to hold Lock.
func (store *Store) Save(session, state string) error {
	if (state != "creating" && state != "starting" && state != "ready") || (state == "creating") != (session == "") {
		return errors.New("invalid session record state")
	}
	if _, _, err := store.Read(); err != nil {
		return err
	}
	if err := store.remember(); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(store.path), ".session-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	writeErr := json.NewEncoder(file).Encode(Record{Version: 1, Ref: store.Ref, Session: session, State: state})
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), store.path)
}

func (store *Store) remember() error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0700); err != nil {
		return err
	}
	if _, err := regular(store.path + ".known"); err != nil {
		return err
	}
	file, err := os.OpenFile(store.path+".known", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return file.Close()
}

func (store *Store) Clear() error {
	if _, err := regular(store.path); err != nil {
		return err
	}
	if err := store.remember(); err != nil {
		return err
	}
	if err := os.Remove(store.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ClearIfSession prevents an old close observer from deleting a replacement's
// association. Disconnects must never call this function.
func (store *Store) ClearIfSession(ctx context.Context, session string) error {
	release, err := store.Lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	record, _, err := store.Read()
	if err != nil {
		return err
	}
	if record == nil || record.Session != session {
		return nil
	}
	return store.Clear()
}

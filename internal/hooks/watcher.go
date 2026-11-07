package hooks

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/itsjustaplant/rexp/internal/sessionstate"
	"github.com/itsjustaplant/rexp/internal/workspace"
)

const Mode = "__watch-close"
const ready = "rexp-hook-ready"
const closed = "rexp-hook-closed"

// Register before inspecting so a close between the two cannot be missed.
// --all keeps the script alive until our own event handler deliberately exits.
// Listening globally does not attach the control stream for session RPCs.
const observer = `local target = rex.args.session
rex.on("session_destroyed", function(ctx, ev)
 if ev.session_id == target then print("rexp-hook-closed"); rex.stop(0) end
end)
local _, attach_err = rex.session.attach{session_id=target}
if attach_err then error("attach close-hook session: " .. tostring(attach_err), 0) end
local view, err = rex.session.view{session_id=target}
if not view then error("inspect close-hook session: " .. tostring(err), 0) end
print("rexp-hook-ready")`

type snapshot struct {
	Version int               `json:"version"`
	Session string            `json:"session"`
	Project workspace.Project `json:"project"`
	Record  *sessionstate.Ref `json:"record,omitempty"`
}

// Watch detaches a worker only after Rex acknowledges its event subscription.
// The worker snapshots the hook, and writes subsequent output to a private log.
func Watch(ctx context.Context, project *workspace.Project, session string) error {
	return watch(ctx, project, session, nil)
}

// WatchManaged also removes the matching session record after actual closure.
func WatchManaged(ctx context.Context, project *workspace.Project, session string, ref sessionstate.Ref) error {
	return watch(ctx, project, session, &ref)
}

func watch(ctx context.Context, project *workspace.Project, session string, ref *sessionstate.Ref) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".rexp", "state", "hooks")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	key := os.Getenv("REX_SERVER") + "\x00" + session
	if ref != nil {
		encoded, _ := json.Marshal(ref)
		key += "\x00" + string(encoded)
	}
	hash := sha256.Sum256([]byte(key))
	path := filepath.Join(dir, fmt.Sprintf("%x.json", hash[:]))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create close observer: %w", err)
	}
	// Store only data needed by this worker, rather than unrelated pane commands.
	saved := workspace.Project{Name: project.Name, Root: project.Root, Env: project.Env, Hooks: workspace.Hooks{AfterClose: project.Hooks.AfterClose}}
	err = json.NewEncoder(file).Encode(snapshot{Version: 1, Session: session, Project: saved, Record: ref})
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
		return err
	}
	success := false
	defer func() {
		if !success {
			os.Remove(path)
		}
	}()
	log, err := os.OpenFile(path+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, Mode, path)
	detach(cmd)
	cmd.Stderr = log
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer pipe.Close()
	if err = cmd.Start(); err != nil {
		return err
	}
	ack := make(chan bool, 1)
	go func() { scanner := bufio.NewScanner(pipe); ack <- scanner.Scan() && scanner.Text() == ready }()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	select {
	case ok := <-ack:
		if ok {
			success = true
			return cmd.Process.Release()
		}
		err = errors.New("close observer failed to subscribe; inspect " + path + ".log")
	case <-ctx.Done():
		err = ctx.Err()
	case <-timer.C:
		err = errors.New("close observer subscription timed out")
	}
	killDetached(cmd)
	cmd.Wait()
	return err
}

// Worker never treats an EOF, server disconnect, or missing session as closure.
func Worker(ctx context.Context, args []string, out, stderr io.Writer) error {
	// Rex's stderr copier and our event reader can write concurrently. Callers
	// may also intentionally supply the same writer for both streams.
	mutex := &sync.Mutex{}
	out = lockedWriter{mutex, out}
	stderr = lockedWriter{mutex, stderr}
	if len(args) != 1 {
		return errors.New("invalid close observer invocation")
	}
	path := args[0]
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("close observer snapshot must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var data snapshot
	decoder := json.NewDecoder(io.LimitReader(file, 4<<20))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&data)
	file.Close()
	if err != nil {
		return err
	}
	if data.Version != 1 || data.Session == "" {
		return errors.New("invalid close observer snapshot")
	}
	defer os.Remove(path)
	params, _ := json.Marshal(map[string]string{"session": data.Session})
	cmd := exec.CommandContext(ctx, "rex", "do", "--all", "--fail-fast", "-e", observer, "--args", string(params))
	cmd.Stderr = stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(pipe)
	subscribed, sawClose := false, false
	for scanner.Scan() {
		switch scanner.Text() {
		case ready:
			if !subscribed {
				subscribed = true
				fmt.Fprintln(out, ready)
			}
		case closed:
			sawClose = true
		default:
			fmt.Fprintln(stderr, scanner.Text())
		}
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if scanErr != nil {
		return scanErr
	}
	if waitErr != nil {
		return fmt.Errorf("Rex close observer: %w", waitErr)
	}
	if !subscribed || !sawClose {
		return errors.New("Rex close observer ended without a session-destroyed event")
	}
	cleanup, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var recordErr error
	if data.Record != nil {
		home, err := os.UserHomeDir()
		if err == nil {
			store, openErr := sessionstate.Open(home, *data.Record)
			err = openErr
			if err == nil {
				err = store.ClearIfSession(cleanup, data.Session)
			}
		}
		recordErr = err
	}
	return errors.Join(recordErr, Run(cleanup, &data.Project, data.Session, "after_close", data.Project.Hooks.AfterClose, stderr, stderr))
}

type lockedWriter struct {
	mutex  *sync.Mutex
	writer io.Writer
}

func (writer lockedWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.writer.Write(data)
}

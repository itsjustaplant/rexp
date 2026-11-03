---
name: rexp
description: Manage saved Rex project workspaces with rexp. Use to create, save, configure, validate, start, use, switch, inspect, stop, or delete rexp workspaces; use Rex directly for individual tabs and panes.
---

# rexp workspaces

`rexp` manages YAML workspace configurations and their Rex sessions. It requires
`rexp` on PATH; live session operations also require `rex` and a reachable server.
Use `rexp <command> --help` for installed flags. Preserve `REX_SERVER` so operations
use the user's Rex server.

## Choose the target

- A name, such as `my-app`, selects `~/.rexp/projects/my-app.yaml` from any directory.
- `.` selects `.rexp.yaml` in the current directory, without ancestor discovery.
- `--file <path>` selects an explicit config. Do not combine it with a name or `.`.
- `rexp list --json` and `rexp search <query> --json` discover global configs only.
  Their `valid` field describes configuration structure, not live session health.

Use the target the user requested. When no target is given, inspect the current
directory's `.rexp.yaml` and global metadata before asking which project they mean.
Do not infer a workspace from the session currently visible in the app.

## Create, capture, and configure

```sh
rexp new my-app              # Starter global config rooted at the current directory
rexp new my-app --local      # Shareable .rexp.yaml with root: "."
rexp save my-app             # Capture the session identified by REX_SESSION
rexp save my-app --session <session-id-or-label>
rexp validate my-app
rexp validate .
```

Run creation or capture from the intended project directory: it becomes the root.
Names start with an ASCII letter or digit, contain only ASCII letters, digits,
hyphens, or underscores, and are at most 64 characters. Existing configs are never
overwritten by `new` or `save`.

Save captures terminal layout, labels, directories, tab selection, and focus. It
cannot recover full commands, environment, hooks, scrollback, or program state.
Do not invent commands from pane titles. `REX_SESSION` identifies the invoking
terminal's session even when the app displays another one; use `--session` when
the user selects a different session.

Inspect the selected YAML before changing it. `rexp edit <target>` opens an
interactive editor and validates after it exits; for direct file edits, run
`rexp validate <target>` afterward. Global roots must be absolute; file configs
resolve relative roots beside the config, and pane `cwd` resolves against root.

Minimal local layout:

```yaml
schema_version: 1
name: my-app
root: "."
tabs:
  - name: Development
    focus: shell
    layout:
      split: columns
      sizes: [50, 50]
      children:
        - pane: shell
        - pane: server
          command: [npm, run, dev]
```

Choose commands from the project and user request. A layout node is either a
`pane` or a `split` with children; `columns` divides width and `rows` divides
height. Sizes are positive integer percentages summing to 100. Pane identifiers
are unique across the workspace. Use `command` for one argv array or `commands`
for a sequence of argv arrays, never both. Shell syntax requires an explicit
shell. String-valued `env` maps merge project → tab → pane. Hooks support
`before_start`, `after_start`, and `after_close`; `before_close` is unsupported.
Keep a built/installed rexp executable available for sequences, env, and hooks.

## Start and inspect

```sh
rexp status my-app --json
rexp start my-app
rexp start . --background
rexp start --file ./config/workspace.yaml --client <client-id-or-label>
```

Starting executes configured commands and hooks when creating a session. Review
unfamiliar configs before launching. A repeated start reuses the existing session
without applying config edits or rerunning commands. Global labels are
`rexp:<name>`; file targets include a path hash, so use the matching file target
instead of constructing a label yourself. Do not rename managed sessions: their
labels establish the association.

`--background` skips app selection and cannot combine with `--client`. With
multiple Rex clients, use the user's intended client or `REX_CLIENT`; do not guess.
Session selection needs Rex Remote Control. Do not assume a flag exists for
opening a session in the current window; check the installed start help.

Status JSON has `running`, `missing`, `incomplete`, or `ambiguous`, all with exit
status 0. Inspect `status`, not just the exit code. It reports session matching,
not service readiness. Verify programs separately if the request needs that.

On a failed start, inspect status before retrying. An incomplete `:starting`
session is preserved with programs possibly running; do not blindly rerun or
delete it. Explain the state and use the matching stop target when the user
authorizes closing it. Ambiguous matches need explicit inspection in Rex.
An app selection error can follow successful setup; retrying a completed start
reuses that session. Report failures without claiming the workspace is ready.

## Switch without starting

For a saved workspace that may not be running, use `rexp use <project>`, `rexp
use .`, or `rexp use --file <path>`. This starts it if needed, then selects it in
the Rex app. It shares `start`'s reuse, hooks, and failure handling; it has no
`--background` option. It selects after startup, without waiting for service health.

`rexp switch <session>` selects an existing session by full ID, exact label, or
global project shorthand (`my-app` matches `rexp:my-app`). It never creates a
session or runs configured programs/hooks. Missing targets are errors; use
`rex ls` to discover live sessions. Full IDs take precedence; ambiguous names
require an explicit ID. Named targets do not require a saved configuration.
`rexp switch .` and `rexp switch --file <path>` resolve a local config's identity.
Use `--client` for a specific Rex app, or preserve `REX_CLIENT`.
An incomplete startup can be selected for inspection by full ID or label, but
switching does not repair it. Shorthand does not select `:starting` sessions.

## Stop versus delete

- `rexp stop <target>` closes the entire live session, including manually added
  panes and running programs, while retaining the YAML and project files.
- `rexp delete <name>` removes only the global YAML; the live session keeps running.
  Local configs are not delete targets.

Use these only when requested or already authorized. For noninteractive execution
after authorization, `--yes` skips the CLI prompt. Stopping the invoking session
requires `--yes` and can terminate your terminal. Do not close a running workspace
merely to apply config edits without the user's authorization.

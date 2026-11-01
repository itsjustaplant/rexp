# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

The source version is currently 0.1.0. `new` creates starter project configurations, `save` captures an existing Rex workspace, `list` shows saved project names, `validate` checks their structure, and `delete` removes saved configurations. Help, version output, and argument parsing are available. `start` launches the configured workspace through the installed Rex CLI, or selects an existing project session. `edit` is not implemented yet.

On startup, rexp creates `.rexp/projects` inside the current user's home directory:

| Platform | Default directory |
| --- | --- |
| macOS | `/Users/<username>/.rexp` |
| Linux | Usually `/home/<username>/.rexp` |
| Windows | Usually `C:\Users\<username>\.rexp` |

The actual path comes from Go's `os.UserHomeDir` (`HOME` on macOS/Linux, `USERPROFILE` on Windows). The setup also runs for help and version invocations. Repeated runs preserve existing files and restore missing directories. On Unix, newly created directories use owner-only permissions. Initialization does not create configuration files yet. `XDG_CONFIG_HOME` does not change this location.

## Development

Requires Go 1.25 or newer. Download dependencies and complete any missing `go.sum` entries first:

```sh
go mod tidy
go run . --help
go run . --version
go run . new --help
```

`go run . --version` prints `rexp 0.1.0`.

Build and check:

```sh
make build
make check
make test
./bin/rexp --help
```

The module path is temporarily `rexp`. Set its final repository URL before distributing it through `go install`.

## Create a project

Create a named configuration from the project's directory:

```sh
cd ~/projects/my-app
rexp new my-app
```

This creates `~/.rexp/projects/my-app.yaml`, recording schema version 1, the current absolute directory as `root`, and a starter tab with a single `layout.pane`. Names must start with an ASCII letter or digit and contain at most 64 ASCII letters, digits, hyphens, or underscores. Windows-reserved names are rejected on all platforms. Existing configurations are never overwritten. New files use owner-only permissions on Unix.

`--edit` (or `-e`) reserves the option to open the new file immediately. Editor launching is deferred: currently the file is created and a notice explains that no editor was opened.

```sh
rexp new my-app --edit
```

## Save a running workspace

Run inside the Rex session you want to capture:

```sh
rexp save my-app
```

This creates `~/.rexp/projects/my-app.yaml` from the session identified by `REX_SESSION`, which is the session containing your terminal, even if the app is displaying another session. To choose explicitly, including when running outside Rex:

```sh
rexp save my-app --session <rex-session-id-or-label>
```

The current directory becomes the configuration's `root`. Each pane gets its observed absolute directory: foreground process first, then the nearest readable ancestor, then the terminal's child process. If no process directory is available, save uses the last reported terminal directory and prints a notice. If no absolute directory can be obtained, saving fails rather than substituting an unrelated path.

Save preserves tab order, tab labels, nested horizontal/vertical splits, and each tab's focused pane. Split fractions are rounded to integer percentages summing to 100; very narrow splits that would round to 0 or 100 are rejected. Invalid or duplicate pane labels are given valid unique identifiers, and blank/duplicate tab names are normalized with notices. The first tab remains the initial tab when starting a saved project; save reports when the source's active tab differs.

**Rex does not expose complete process arguments.** Saved panes open interactive shells; add `command` arrays manually before starting if you want to launch programs. Saving does not capture terminal scrollback, shell history, environment changes, editor buffers, or running process state. It does not replay commands inferred from a title or process name.

Only terminal panes in one tiled layer per tab are supported. Floating/multiple layers, other block types, and panes on different hosts are rejected. Paths are native to the machine running rexp; remote workspace restoration is outside the current scope. The session is inspected again before writing, and changes to its layout, labels, or focus cause a retry error. Directory observations are collected sequentially, not as an atomic process snapshot.

Existing files, directories, and symbolic links at the destination are never overwritten. Use a new project name to save another snapshot. Saving only reads Rex state and writes the new configuration; it does not rename, close, or otherwise modify the source session.

```sh
rexp save saved-workspace
rexp validate saved-workspace
rexp start saved-workspace
```

Starting creates a session named `rexp:saved-workspace`, or reuses one already carrying that label. Use a fresh name when testing restoration so reuse does not hide changes.

## Layout configuration

```yaml
schema_version: 1
name: my-app
root: /absolute/path/to/my-app
tabs:
  - name: Development
    focus: editor
    layout:
      split: columns
      sizes: [50, 50]
      children:
        - pane: editor
          command: [nvim, .]
        - split: rows
          sizes: [34, 33, 33]
          children:
            - pane: server
              cwd: ./backend
              command: [npm, run, dev]
            - pane: logs
              command: [tail, -f, ./logs/app.log]
            - pane: shell
              command: []
```

`columns` arranges children left to right and divides width; `rows` arranges them top to bottom and divides height. `sizes` contains positive integer percentages summing to 100, one per child. Omit `sizes` for equal shares. Groups may nest, with at least two children per split. Each pane fills its parent's other dimension. Rex's CLI accepts integer split percentages, so some proportions are approximate; terminal cells and minimum pane sizes can also affect the result. Layouts requiring ratios outside Rex's 1–99 range are rejected before creation.

Each layout node defines either `pane` or `split`. Split nodes accept only `split`, `sizes`, and `children`; panes accept only `pane`, `command`, and `cwd`. Pane identifiers follow project-name rules and must be unique across the project. Tab names must be nonempty and unique. `focus` must name a pane within its tab; omit it to focus the first pane. The first tab is initially selected on creation. Reusing an existing session preserves its current layout and focus.

Pane `cwd` defaults to `root`, and relative values resolve against `root`. Root must be absolute; use native paths for your OS. `command` is an array of strings; omitted or empty opens an interactive shell. Shell syntax needs an explicit shell, such as `[sh, -c, "npm run build && npm test"]`.

## Validate a configuration

```sh
rexp validate my-app
```

Checks YAML syntax, schema version, names, node types, percentages, focus references, and command types. Unknown fields, duplicate YAML keys, aliases, null values, and multiple YAML documents are rejected. Layout nesting is limited to 32 levels. No commands run, and no Rex connection is made. This checks configuration structure, not whether paths/programs exist or the Rex server supports a particular layout.

See [the four-pane example](examples/four-panes.yaml). Existing files created with the previous `tabs[].panes` template are not automatically rewritten: add `schema_version: 1` and replace each tab's `panes` with a `layout` tree. `list` and `delete` still work on those filenames without parsing them.

## List projects

```sh
rexp list
```

Prints saved project names alphabetically (case-sensitive order), one per line, without the `.yaml` extension. An empty list shows how to create a project. Listing uses regular `.yaml` filenames with valid project names; other files, directories, and symbolic links are ignored. YAML contents are not read or validated, so a manually edited invalid YAML file still appears in the list. No Rex connection is required.

## Delete a project configuration

```sh
rexp delete my-app
```

The command shows the configuration's path and asks for confirmation. Enter `y` or `yes` to delete it; other answers cancel. With no input available, the command returns an error and keeps the file. To skip confirmation:

```sh
rexp delete my-app --yes
# Short form:
rexp delete my-app -y
```

Only `~/.rexp/projects/my-app.yaml` is removed. The project's source directory, other configurations, and running Rex sessions are left alone. Missing configurations, invalid names, directories, and symbolic links are rejected. Deleted configurations are not moved to a trash folder.

## Start a workspace

Launch a saved configuration from any directory:

```sh
rexp start my-app
rexp start my-app --background
rexp start my-app --client <rex-client-id-or-label>
```

Requires the Rex CLI on `PATH` and a reachable Rex server. `start` validates the entire configuration and checks all configured directories before creating terminals. This initial implementation is intended for local workspaces; it inherits Rex's server selection environment (`REX_SERVER`). Directory checks run on the machine running rexp.

Each project uses a session named `rexp:<project>`. A subsequent `start` selects that session without rerunning commands or applying configuration edits. Duplicate labels, labels differing only in case, and incomplete startups return errors. Session labels identify workspaces: renaming the session breaks this association. Avoid running simultaneous starts for the same project.

Rexp creates tabs and nested splits using returned Rex IDs, passes commands as argument arrays, labels each pane, and sets each tab's configured focus. Commands run once when their panes are created. Rex owns the terminals after rexp exits. Panes remain open after commands exit so their output is visible; successfully creating a pane does not mean its application is healthy. There is no readiness checking or dependency ordering between pane commands.

Selecting a session in the app requires Rex Remote Control. With multiple clients, use `--client` (or Rex's `REX_CLIENT` environment). `--background` skips app selection; it cannot be combined with `--client`. A selection error can occur after successful setup: retrying reuses the completed session.

During creation, the session is named `rexp:<project>:starting`. If setup fails, it remains under that name with existing panes running. Further starts stop with an error rather than rerunning commands. Inspect the session in Rex; close it manually before retrying, or finish it yourself and rename it to `rexp:<project>`. Rexp does not automatically delete sessions or undo commands after a failure.

For a layout-only smoke test, use [the four-shell example](examples/four-shells.yaml), set `root` to an existing absolute directory, and save it as `~/.rexp/projects/four-shells.yaml`:

```sh
rexp validate four-shells
rexp start four-shells
rexp start four-shells
```

Expect one pane covering the left half and three stacked panes covering the right half, with the left pane focused. The second start prints `Reused` and creates no additional panes. Editing remains planned through `rexp edit my-app`.

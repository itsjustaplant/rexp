# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

The source version is currently 0.1.0. `new` creates starter project configurations, `list` shows saved project names, `validate` checks their structure, and `delete` removes saved configurations. Help, version output, and argument parsing are available. `edit` and `start` currently return explicit not-implemented errors. No Rex sessions are created yet.

On startup, rexp creates `.rexp/projects` inside the current user's home directory:

| Platform | Default directory |
| --- | --- |
| macOS | `/Users/<username>/.rexp` |
| Linux | Usually `/home/<username>/.rexp` |
| Windows | Usually `C:\Users\<username>\.rexp` |

The actual path comes from Go's `os.UserHomeDir` (`HOME` on macOS/Linux, `USERPROFILE` on Windows). The setup also runs for help and version invocations. Repeated runs preserve existing files and restore missing directories. On Unix, newly created directories use owner-only permissions. Initialization does not create configuration files yet. `XDG_CONFIG_HOME` does not change this location.

## Development

Requires Go 1.25 or newer. Download dependencies and generate `go.sum` first:

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

`columns` arranges children left to right and divides width; `rows` arranges them top to bottom and divides height. `sizes` contains positive integer percentages summing to 100, one per child. Omit `sizes` for equal shares. Groups may nest, with at least two children per split. Each pane fills its parent's other dimension. Actual terminal sizes will depend on cell rounding and minimum sizes when launching is implemented.

Each layout node defines either `pane` or `split`. Split nodes accept only `split`, `sizes`, and `children`; panes accept only `pane`, `command`, and `cwd`. Pane identifiers follow project-name rules and must be unique across the project. Tab names must be nonempty and unique. `focus` must name a pane within its tab; omit it to focus the first pane. The first tab will initially be selected when launch support is added.

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

## Planned launch workflow

Then launch the workspace from any directory:

```sh
rexp start my-app
```

The intended behavior is to create the configured Rex session or select the existing project session. Editing uses `rexp edit my-app`; saved projects are listed with `rexp list`.

Launching and editing are planned and are not available yet.

# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

The source version is currently 0.0.1. `new` creates starter project configurations, `list` shows saved project names, and `delete` removes saved configurations. Help, version output, and argument parsing are available. `edit` and `start` currently return explicit not-implemented errors. No Rex sessions are created yet.

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

`go run . --version` prints `rexp 0.0.1`.

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

This creates `~/.rexp/projects/my-app.yaml`, recording the current absolute directory as `root` and a starter tab with one shell pane. Names must start with an ASCII letter or digit and contain at most 64 ASCII letters, digits, hyphens, or underscores. Windows-reserved names are rejected on all platforms. Existing configurations are never overwritten. New files use owner-only permissions on Unix.

`--edit` (or `-e`) reserves the option to open the new file immediately. Editor launching is deferred: currently the file is created and a notice explains that no editor was opened.

```sh
rexp new my-app --edit
```

The generated YAML layout is a starter schema; launching it will be implemented with `start`.

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

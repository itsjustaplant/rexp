# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

Version 0.0.1 is the initial Go/Cobra scaffold. Help, version output, argument parsing, and the command structure are wired up. `init`, `edit`, `start`, and `list` currently return explicit not-implemented errors. No configuration files or Rex sessions are created.

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
go run . init --help
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

## Planned workflow

Create a named configuration from the project's directory:

```sh
cd ~/projects/my-app
rexp init my-app
```

The intended configuration location is `~/.rexp/projects/my-app.yaml` (inside the user's home directory on every platform). The future `init` command will save the project directory and open the user's editor for configuring tabs, panes, and commands.

Then launch the workspace from any directory:

```sh
rexp start my-app
```

The intended behavior is to create the configured Rex session or select the existing project session. Editing uses `rexp edit my-app`; saved projects are listed with `rexp list`.

These project operations are planned and are not available in this scaffold.

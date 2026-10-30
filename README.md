# rexp

Named project workspaces for [Rex](https://www.superlogical.com/rex).

## Status

Version 0.0.1 is the initial Go/Cobra scaffold. Help, version output, argument parsing, and the command structure are wired up. `init`, `edit`, `start`, and `list` currently return explicit not-implemented errors. No configuration files or Rex sessions are created.

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

The intended configuration location is `$XDG_CONFIG_HOME/rexp/projects/my-app.yaml`, falling back to `~/.config/rexp/projects/my-app.yaml`. Initialization will save the project directory and open the user's editor for configuring tabs, panes, and commands.

Then launch the workspace from any directory:

```sh
rexp start my-app
```

The intended behavior is to create the configured Rex session or select the existing project session. Editing uses `rexp edit my-app`; saved projects are listed with `rexp list`.

These project operations are planned and are not available in this scaffold.

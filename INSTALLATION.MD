# Install rexp

Rexp creates project workspaces in [Rex](https://www.superlogical.com/rex). Install it using Go, download a prebuilt binary, or build from a source checkout.

The commands and asset names below describe the release setup. Versioned `go install` and binary downloads become available after a release-ready version tag and its artifacts are published. Until then, use the source-checkout method.

## License

Rexp is source-available under [MIT with Commons Clause v1.0](LICENSE). Internal commercial use and modifications are allowed subject to the license; selling rexp or qualifying paid products/services based on it is restricted. Review the full terms before redistribution.

## Requirements

- [Rex and its CLI](https://www.superlogical.com/rex/docs/installation) for workspace commands such as start, save, status, and stop. Installing rexp does not install Rex.
- [Go 1.25 or newer](https://go.dev/doc/install) for `go install` or source builds. Prebuilt rexp binaries do not require Go.

Rexp has been live-tested on macOS. Linux and Windows builds are cross-compiled; they still need live verification. Check Rex's guide for its current platform availability.

## Option 1: Install with Go

After the first compatible version is published:

```sh
go install github.com/itsjustaplant/rexp@latest
```

Go downloads the source, builds rexp, and installs the executable. This command does not use the prebuilt GitHub release assets. To install a specific release, replace `latest` with its version tag, such as `vX.Y.Z`.

### Make the command available on PATH

Installation does not edit `.bashrc` or `.zshrc`. With Go's default settings, the executable is installed in `$HOME/go/bin` on macOS/Linux or `%USERPROFILE%\go\bin` on Windows.

For Bash/Zsh with default Go settings:

```sh
export PATH="$HOME/go/bin:$PATH"
rexp --version
rexp help
```

Add that PATH line to your `~/.zshrc` or `~/.bashrc` yourself to keep it in new terminals. If your Go settings are customized, run `go env GOBIN GOPATH`: use GOBIN when set, otherwise the `bin` directory under the first GOPATH entry. Go does not require installing into the source checkout.

In PowerShell with default Go settings:

```powershell
$rexpBin = Join-Path $HOME 'go\bin'
$env:Path = "$rexpBin;$env:Path"
rexp --version
rexp help
```

For new Windows terminals, add that directory to your **user** Path through Environment Variables settings, then reopen the terminal. Respect a custom GOBIN/GOPATH as above.

## Option 2: Download a prebuilt binary

When release assets are published, open [GitHub Releases](https://github.com/itsjustaplant/rexp/releases), choose a version, and download the archive matching your system plus `checksums.txt`.

| System | Archive name (`<version>` is the release number without `v`) |
| --- | --- |
| macOS, Apple Silicon | `rexp_<version>_darwin_arm64.tar.gz` |
| macOS, Intel | `rexp_<version>_darwin_amd64.tar.gz` |
| Linux, x86-64 | `rexp_<version>_linux_amd64.tar.gz` |
| Linux, ARM64 | `rexp_<version>_linux_arm64.tar.gz` |
| Windows, x86-64 | `rexp_<version>_windows_amd64.zip` |
| Windows, ARM64 | `rexp_<version>_windows_arm64.zip` |

Before extracting, calculate the archive's SHA-256 hash and compare it with the matching line in `checksums.txt`. Replace `<archive-file>` with the downloaded filename:

```sh
# macOS:
shasum -a 256 "<archive-file>"

# Linux:
sha256sum "<archive-file>"
```

```powershell
# Windows:
Get-FileHash -Algorithm SHA256 -Path '<archive-file>'
```

### macOS and Linux

Extract the archive. From the extracted directory containing `rexp`:

```sh
mkdir -p "$HOME/.local/bin"
install -m 755 ./rexp "$HOME/.local/bin/rexp"
export PATH="$HOME/.local/bin:$PATH"
rexp --version
rexp help
```

No administrator privileges are required. Add the PATH line to your shell startup file yourself for new terminals.

### Windows (PowerShell)

Extract the ZIP. From the extracted directory containing `rexp.exe`:

```powershell
$rexpBin = Join-Path $env:LOCALAPPDATA 'rexp\bin'
New-Item -ItemType Directory -Force -Path $rexpBin | Out-Null
Copy-Item .\rexp.exe (Join-Path $rexpBin 'rexp.exe')
$env:Path = "$rexpBin;$env:Path"
rexp --version
rexp help
```

Add `%LOCALAPPDATA%\rexp\bin` to your **user** Path through Windows' Environment Variables settings for new terminals.

## Option 3: Build from a source checkout

Clone [the repository](https://github.com/itsjustaplant/rexp), or use **Code → Download ZIP**, extract it, and open a terminal in the folder containing `go.mod`.

```sh
git clone https://github.com/itsjustaplant/rexp.git
cd rexp
go mod download
go install .
```

For a ZIP download, skip the clone/cd commands and run the last two commands in the extracted source directory. Follow the Go PATH setup above. This installs the version in your checkout; it does not require a published tag.

For a build you can run directly from the checkout:

```sh
# macOS/Linux:
go build -o bin/rexp .
./bin/rexp --version
```

```powershell
# Windows:
go build -o bin/rexp.exe .
.\bin\rexp.exe --version
```

## Set up skills, completion, and an example

Once rexp is on PATH, run `rexp` in an interactive terminal to start first-run
setup, or run it explicitly:

```sh
rexp init
```

It asks before installing skills for Codex or Claude Code and before adding shell
completion. It then offers a two-pane demo with a live clock and the rexp dinosaur.
Use `rexp init --no-demo` to set up integrations only, or `--shell zsh`, `--shell
bash`, or `--shell fish` to override shell detection. Open a new shell afterward
with rexp on PATH. With several Rex app clients, use `--client <client>` for the
demo. Setup can be replayed with `rexp init` or `rexp intro`; repeated setup keeps
existing integrations and reuses the demo session.

Only running bare `rexp` in an interactive terminal starts onboarding
automatically; installation, help/version, scripts, and completion probes do not.
See [first-run setup](README.md#first-run-setup) for file locations and details.

## Verify your first workspace

Check that Rex is available:

```sh
rex --help
```

From your project's directory inside Rex:

```sh
rexp new my-app --local
rexp validate .
rexp start .
rexp status .
```

`new --local` creates `.rexp.yaml` and refuses to overwrite an existing file. The starter config opens one shell pane. Use `rexp edit .` to configure tabs, splits, commands, and environment variables; see the [README](README.md) for examples.

Alternatively, `rexp new my-app` saves a named config under `~/.rexp/projects`; `rexp start my-app` opens it from any directory.

Ordinary invocations initialize `~/.rexp/projects` (`%USERPROFILE%\.rexp\projects` on Windows). Completion generation skips initialization. Your configuration files are separate from the installed executable and remain in place during updates.

With multiple Rex clients, choose one using `rexp start . --client <client>` or use `--background` to avoid switching the app.

## Optional shell completion

PATH makes `rexp help` callable. Completion is a separate, optional step for Tab suggestions. Load the script for your shell; rexp does not edit startup files.

### Zsh

If completion is already initialized in your shell setup, only the final line is needed:

```sh
autoload -Uz compinit
compinit
source <(rexp completion zsh)
```

### Bash

With bash-completion available in your shell:

```sh
source <(rexp completion bash)
```

### Fish

```fish
rexp completion fish | source
```

### PowerShell

```powershell
rexp completion powershell | Out-String | Invoke-Expression
```

For persistent completion, use `rexp init` or add the appropriate load command to your shell startup file after PATH/completion initialization. Suggestions do not run project code or contact Rex.

## Update or roll back

Use the same installation method to update. Updates are manual; rexp does not update itself or automatically check for new versions.

- **Go:** rerun `go install github.com/itsjustaplant/rexp@latest`. To select an earlier version, install its specific tag instead.
- **Prebuilt binary:** download the desired release for your platform, verify its checksum, and repeat the binary installation steps.
- **Git checkout:** run `git pull --ff-only`, `go mod download`, and `go install .`. For a ZIP checkout, download a fresh source ZIP instead.

Check `rexp --version` afterward and reload completion when commands/flags change. Use `command -v rexp` on macOS/Linux or `Get-Command rexp` in PowerShell if an older binary appears first on PATH.

Close workspace sessions before replacing the executable: pane runners and close-hook observers use the installed rexp binary. Update only the executable; keep your project directories and `.rexp` configurations.

## Troubleshooting

- **Command not found:** confirm your Go or binary installation directory is on PATH, then reopen the terminal. Try the full executable path to distinguish an installation problem from a PATH problem.
- **`Rex CLI not found on PATH`:** finish Rex's CLI installation, reopen the terminal, and check `rex --help`.
- **Rex connection errors:** run `rex whoami --json` in the same terminal and confirm the intended server is reachable.
- **Go download errors:** check network access, repository access, and whether the requested version has been published. Source builds must run beside `go.mod`.
- **Config already exists:** use `rexp edit .` or `rexp start .` for that existing local config.

Homebrew installation and an automatic updater are not configured yet.

# internal/android — Android/Termux Bootstrap Package

This package provides a **first-run bootstrap** for `arduino-cli` running natively on Android/Termux without chroots, PRoot, Docker, or a Linux distribution layer.

## Problem

The upstream `arduino-cli` assumes a conventional Linux filesystem layout:

| Assumption | Reality on Android/Termux |
|---|---|
| Writable home at `/home/<user>` | Home is at `$HOME` = `/data/data/com.termux/files/home` |
| System prefix at `/usr` | Termux prefix is `$PREFIX` = `/data/data/com.termux/files/usr` |
| Temp at `/tmp` | `/tmp` is often `noexec`; use `$PREFIX/tmp` |
| Config written to `~/.arduino15` | Must use a path under the Termux home |

If `arduino-cli` starts with its upstream defaults it will immediately fail with permission errors or "no such file or directory" because `/home` and `/usr` do not exist in the Android filesystem.

## Solution

This package:

1. **Detects** the Termux environment (`$PREFIX`, `$HOME`).
2. **Creates** `$HOME/.arduino15-android/` and all required sub-directories.
3. **Writes** a valid `arduino-cli.yaml` that points all directory keys to writable, executable-friendly paths.
4. **Validates** existing configs for Android-incompatible (hardcoded `/usr` or `/home`) path references.
5. **Patches** upstream-generated configs to replace hardcoded paths in-place.

## Directory Layout

After bootstrap:

```
$HOME/.arduino15-android/
├── arduino-cli.yaml          ← generated config
├── staging/
│   └── packages/             ← downloads directory
└── sketchbook/               ← user/sketchbook directory

$PREFIX/tmp/arduino-cli/      ← temp directory (on exec-friendly fs)
```

## API

| Symbol | Description |
|---|---|
| `Bootstrap(ctx)` | Main entry point. Idempotent. |
| `DetectTermuxEnvironment(ctx)` | Resolve Termux paths from the environment. |
| `ValidateConfig(path)` | Check a YAML config for Android incompatibilities. |
| `PatchConfigPaths(path, ctx)` | Rewrite hardcoded `/usr`/`/home` paths in-place. |
| `NewBootstrapCmd()` | Returns a `cobra.Command` for the `android-bootstrap` sub-command. |
| `AddFlags(cmd)` | Registers `--android-bootstrap` and `--android-bootstrap-force` flags. |
| `RunFromFlags(cmd, cfgPath)` | Called from `PersistentPreRunE`; no-op unless flags are set. |

## CLI Integration

Wire the bootstrap into the root `cobra.Command` by calling from `internal/cli/`:

```go
cli.RegisterAndroidBootstrap(rootCmd, func() string {
    return resolvedConfigFilePath
})
```

This registers:
- `--android-bootstrap` — auto-bootstrap on first run (idempotent).
- `--android-bootstrap-force` — force re-generation of the config.
- `android-bootstrap` sub-command — explicit one-shot bootstrap.

## Running the Bootstrap

```sh
# One-shot explicit bootstrap:
arduino-cli android-bootstrap

# Force regeneration:
arduino-cli android-bootstrap --force

# Auto-bootstrap wired into any command:
arduino-cli --android-bootstrap version
```

## Tests

Unit tests:

```sh
go test ./internal/android/...
```

Integration tests (no device required — uses a sandboxed fake Termux tree):

```sh
go test ./internal/integrationtest/android/...
```

# Altered source: go-shirei's build tools

`mullion-web`, `mullion-mobilerun` and `mullion-bundle` are altered versions of
go-shirei's `cmd/shirei_web`, `cmd/shirei_mobilerun` and `cmd/shirei_bundle`.
They are not the original software.

- Project: <https://github.com/hasenj/go-shirei>, module `go.hasen.dev/shirei`
- Merged: commit `26d6369dce19fc82643f4f403d2dd87907f85437` (v0.9.1, 2026-11-16)
- Copyright (c) 2026 Judi Systems, under the zlib licence, whose text is
  [`COPYING`](COPYING) beside this file and
  [`../licenses/LICENSE.shirei`](../licenses/LICENSE.shirei)

go-shirei's authors did not write the alterations described here and are not
responsible for them. The library these tools build for is
[`../native`](../native/ALTERED.md).

## How they were altered

1. **Module and directories.** Imports were rewritten as in
   `../native/ALTERED.md` item 1, and each tool's directory was renamed
   (`shirei_web` to `mullion-web`, `shirei_mobilerun` to `mullion-mobilerun`,
   `shirei_bundle` to `mullion-bundle`).
2. **The name in one place.** The project name, its title, the environment
   variable prefix, the reverse-DNS prefix of application ids, the Android glue
   package and class, the logcat tag, the capability file name, log prefixes and
   temporary directory patterns come from `../internal/brand` (written for
   mullion) instead of being spelled out where they are used.
3. **The Android glue class** is `dev.mullion.host.MullionActivity`
   (`embed/MullionActivity.java` in `mullion-bundle` and `mullion-mobilerun`),
   and the JNI symbols in `../native/androidbackend/jni.c` are named to match.
   The Xcode scheme of `mullion-mobilerun`'s embedded iOS host is
   `mullion.xcscheme`.
4. **Main packages are found with `go list`.** `packages.go` in `mullion-bundle`
   and `mullion-mobilerun` asks the toolchain for the module's main packages
   where the original text-searched the tree for `func main`, which also removed
   the `go.hasen.dev/textsearch` dependency.
5. **Module paths in the text** of the READMEs and of
   `mullion-mobilerun/embed/ios-run.sh` (a comment and its usage text) are this
   tree's.
6. **Not carried.** `shirei_tester` and `behavior_runner`, which run go-shirei's
   behaviour programs; those programs were not carried either.

The note on reading the text in `../native/ALTERED.md` applies here as well:
where the tools' comments and READMEs say "mullion", they describe what the
tools are now; who wrote them is go-shirei's authors.

## The files

Each file compared with its counterpart at the merged commit, after item 1 and
the name replacement are applied to the upstream text:

| directory | files | unchanged | rewrite only | changed further |
|---|---:|---:|---:|---:|
| `mullion-bundle/` | 29 | 13 | 6 | 10 |
| `mullion-mobilerun/` | 25 | 5 | 8 | 12 |
| `mullion-web/` | 7 | 2 | 2 | 3 |
| **total** | **61** | **20** | **16** | **25** |

No file here was added for mullion, and the counts leave out this file and
`COPYING`.

Changed further (items 2 to 5):

- `mullion-bundle/`: `android.go`, `android_adb.go`, `config.go`, `desktop.go`,
  `gui.go`, `ios.go`, `macos.go`, `main.go`, `packages.go`, `provision.go`
- `mullion-mobilerun/`: `android_build.go`, `android_embed.go`,
  `capabilities.go`, `config.go`, `config_test.go`, `embed/ios-run.sh`,
  `gui.go`, `ios_embed.go`, `ios_run.go`, `ios_sims.go`, `main.go`,
  `packages.go`
- `mullion-web/`: `compiler.go`, `gallery.go`, `main.go`

`mullion-web/wasm_exec.js` is the Go Authors' file, unchanged, under the BSD
licence in [`../licenses/LICENSE.golang`](../licenses/LICENSE.golang).

# mullion-mobilerun — development installs for iOS and Android

Build and launch a Mullion `package main` on a simulator, emulator, or USB
device while iterating. Android uses a **debug** keystore and debuggable APKs;
iOS uses simulator or development signing.

For **release** IPAs and APKs, use **`mullion-bundle`**:
[../mullion-bundle/README.md](../mullion-bundle/README.md).

## Install

```bash
go install go.hasen.dev/mullion/cmd/mullion-mobilerun@latest
```

Put `$(go env GOPATH)/bin` (or `GOBIN`) on your `PATH`, then:

```bash
mullion-mobilerun
```

From a local checkout of this module:

```bash
go install ./cmd/mullion-mobilerun
# or:
go run ./cmd/mullion-mobilerun
```

## Host setup

Follow the full environment guides:

- [Android](../../docs/android.md) — JDK, SDK, NDK, `adb`
- [iOS](../../docs/ios.md) — macOS + Xcode (Simulator and/or device)

## Usage

### GUI

```bash
mullion-mobilerun
```

Pick platform, package, optional id/name/icon, then run.

### CLI

```bash
# Android device / emulator
mullion-mobilerun -platform android ./path/to/package

# Build APK only (no install)
mullion-mobilerun -platform android -build-only ./path/to/package

# iOS Simulator
mullion-mobilerun -platform ios ./path/to/package

# Physical iPhone (Team ID required)
mullion-mobilerun -platform ios -device -team YOURTEAMID ./path/to/package
```

Preferences: `<UserConfigDir>/mullion/mobile-run.json`.

## Related

- [docs/android.md](../../docs/android.md)
- [docs/ios.md](../../docs/ios.md)
- [docs/mobile-extensions.md](../../docs/mobile-extensions.md) — camera and capability tokens
- [mullion-bundle](../mullion-bundle/README.md) — release packaging

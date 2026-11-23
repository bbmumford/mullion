# camera

Reference Mullion **extension**: take a still photo with the system camera UI
and produce `*image.RGBA` for `native.UseImage` — without putting camera
support in core.

Module: `github.com/bbmumford/mullion/native/ext/camera`

## Run the demo

```sh
mullion-mobilerun -platform ios ./mullion/demos/mobile-camera
mullion-mobilerun -platform ios -device ./mullion/demos/mobile-camera
mullion-mobilerun -platform android ./mullion/demos/mobile-camera
```

mullion-mobilerun reads `mullion.capabilities` from this module (`camera`, `photos`)
and injects matching Info.plist purpose strings / Android permissions.

## Mechanism

See **[Mobile extensions and escape hatches](../../docs/mobile-extensions.md)**
for the full design (`EscapeHatchBackendContext`, **per-backend context
types**, capabilities file, token catalogue). This package is the worked
example in that guide.

```
TakePhoto
  → GetHost().EscapeHatchBackendContext
  → iOS / Android: system camera / gallery
  → desktop: Done(errUnsupported)
  → *image.RGBA → Done(err) when supported
```

| Platform | Context / host | How the camera handler uses it |
|----------|----------------|--------------------------------|
| iOS | `iosbackend.Context` | `RootViewController()` → UIImagePicker in this package |
| Android | `androidbackend` generic bridges | `OnActivityResult` / `StartActivityForResult` + camera JNI here |
| Desktop | cocoa / win32 / x11 / wayland | Stub reports unsupported |

Camera-specific code lives **only** in this package on mobile platforms.

## App usage

```go
var photo *image.RGBA

if Button(SymImage, "Take photo") {
    camera.TakePhoto(&photo, func(err error) {
        native.WithFrameLock(func() {
            if err == nil && photo != nil {
                native.UseImage("camera/last", photo)
            }
            native.RequestNextFrame()
        })
    })
}
```

## Files

| File | Role |
|------|------|
| `camera.go` | Portable `TakePhoto` API |
| `camera_ios.go` / `.m` | UIImagePicker + RGBA |
| `camera_android.go` / `camera_jni_android.c` | Camera/gallery via generic androidbackend bridges |
| `camera_pending.go` | One in-flight capture slot |
| `mullion.capabilities` | `camera` + `photos` for mullion-mobilerun |
| `../demos/mobile-camera` | Button + image view (iOS + Android) |

# ioshost

The UIKit host template used by `mullion-mobilerun` lives at:

```
mullion/cmd/mullion-mobilerun/embed/ioshost/
```

That tree is **embedded** into the `mullion-mobilerun` binary and extracted to
the user cache on demand. Do not put per-app names or `libmullion.a` here — the
build always copies the template to a scratch `host-work` directory first.

Edit the template under `cmd/mullion-mobilerun/embed/ioshost/`, then rebuild:

```
go install github.com/bbmumford/mullion/cmd/mullion-mobilerun@latest
```

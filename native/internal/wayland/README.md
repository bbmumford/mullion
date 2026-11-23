# internal/wayland

Wayland protocol bindings, vendored from github.com/neurlang/wayland
v0.4.3 (MIT — see LICENSE) and modified for go-shirei, by go-shirei's
authors. Upstream is a UI library; go-shirei needed only its protocol
layer, and owning the code let the event loop grow what its backend needs (deadline-based dispatch,
compositor flush provocation — see wl/context.go).

Local changes vs upstream:
- import paths rewritten into go-shirei's module (in this tree,
  github.com/bbmumford/mullion/native/internal/wayland/...)
- yalue/native_endian replaced with encoding/binary's LittleEndian
  (every mullion target is little-endian)
- wl.Context gains RunTimeout (read-deadline dispatch); readEvent clears
  the deadline after the header read so a timeout can never split an
  event mid-read
- cursorshape/ is go-shirei's own package (not from upstream): the staging
  wp-cursor-shape-v1 protocol, originally hand-written in the backend
- wlcursor/ + wlcursor/xcursor/ (the cursor-theme loader) have the
  swizzle assembly deleted outright: Xcursor pixels are already wl_shm
  ARGB8888 little-endian, so the raw bytes pass through — upstream's
  R<->B swap both crashed on arm64 (hand-written NEON) and shipped
  color-swapped cursors (invisible on grayscale arrows). xcursor's
  parser is rewritten (bounds-checked, single Pix slice) and
  nearestImages actually picks the nearest size (ties to the larger),
  instead of exact-match-or-first

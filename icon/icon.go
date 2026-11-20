// Package icon carries an application's artwork between the surfaces that need
// it.
//
// The three trees mullion is built from each asked for an icon in their own
// shape: the tray wanted PNG bytes plus separate dark-mode and macOS template
// variants, the Go-drawn surface wanted a file path or a decoded image.Image,
// and the webview surface wanted PNG bytes for the dock. An application should
// state its icon once, so Icon holds every variant and each surface takes the
// part it can use.
package icon

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // Image accepts whatever the application already decoded
	"image/png"
	"os"
)

// An Icon is one piece of application artwork and its platform variants. The
// zero Icon is empty, which every surface reads as "no icon": that is a
// supported state, not an error, so an application can leave it out.
type Icon struct {
	// Data is the icon as PNG bytes. Every surface can take this one.
	Data []byte

	// Dark is an alternative used where the shell is dark and the system does
	// not adapt the icon itself -- the Windows notification area. Empty means
	// Data is used in both themes.
	Dark []byte

	// Template marks Data as a macOS template image: AppKit ignores its colour
	// and tints the shape to match the menu bar, light or dark. Set it for a
	// flat monochrome silhouette, leave it off for a full-colour icon, and
	// note that it makes a colour icon come out as a solid block.
	Template bool
}

// PNG is an Icon from PNG bytes, which is what an embedded asset usually is.
// The bytes are not copied or decoded; a malformed PNG surfaces later, when a
// surface hands it to the platform.
func PNG(data []byte) Icon { return Icon{Data: data} }

// File is an Icon read from a PNG on disk.
func File(path string) (Icon, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Icon{}, fmt.Errorf("icon: %w", err)
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		return Icon{}, fmt.Errorf("icon: %s is not a PNG: %w", path, err)
	}
	return Icon{Data: data}, nil
}

// Image is an Icon encoded from an already decoded image, for a program that
// draws or resizes its own artwork.
func Image(img image.Image) (Icon, error) {
	if img == nil {
		return Icon{}, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return Icon{}, fmt.Errorf("icon: encoding: %w", err)
	}
	return Icon{Data: buf.Bytes()}, nil
}

// WithDark returns a copy carrying a dark-theme variant.
func (i Icon) WithDark(data []byte) Icon {
	i.Dark = data
	return i
}

// AsTemplate returns a copy marked as a macOS template image.
func (i Icon) AsTemplate() Icon {
	i.Template = true
	return i
}

// Empty reports whether there is no artwork to install. Surfaces check it
// instead of treating a missing icon as a failure.
func (i Icon) Empty() bool { return len(i.Data) == 0 }

// Decode returns the icon as an image, for the surfaces that set a window or
// dock icon from a decoded image rather than from bytes.
func (i Icon) Decode() (image.Image, error) {
	if i.Empty() {
		return nil, fmt.Errorf("icon: empty")
	}
	img, _, err := image.Decode(bytes.NewReader(i.Data))
	if err != nil {
		return nil, fmt.Errorf("icon: decoding: %w", err)
	}
	return img, nil
}

// ForTheme picks the bytes to install for a shell that reports itself as dark.
// It is what a tray backend calls when the system tells it the theme changed.
func (i Icon) ForTheme(dark bool) []byte {
	if dark && len(i.Dark) > 0 {
		return i.Dark
	}
	return i.Data
}

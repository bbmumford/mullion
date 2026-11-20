package icon

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xff, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding the fixture: %v", err)
	}
	return buf.Bytes()
}

// An empty Icon is a supported state, not a failure: an application may run
// with no artwork and every surface reads it as "no icon".
func TestZeroIconIsEmpty(t *testing.T) {
	var ic Icon
	if !ic.Empty() {
		t.Error("the zero Icon does not report itself empty")
	}
	if ic.ForTheme(true) != nil || ic.ForTheme(false) != nil {
		t.Error("the zero Icon offered bytes to install")
	}
}

func TestPNGKeepsTheBytesItWasGiven(t *testing.T) {
	data := testPNG(t)
	ic := PNG(data)
	if ic.Empty() {
		t.Fatal("an Icon built from PNG bytes reports itself empty")
	}
	if !bytes.Equal(ic.Data, data) {
		t.Error("the bytes were altered on the way in")
	}
	if ic.Template {
		t.Error("PNG marked the icon as a macOS template")
	}
}

func TestForThemePicksTheDarkVariantOnlyWhenThereIsOne(t *testing.T) {
	light, dark := testPNG(t), append(testPNG(t), 0)
	plain := PNG(light)
	if got := plain.ForTheme(true); !bytes.Equal(got, light) {
		t.Error("with no dark variant, a dark shell should still get the one icon")
	}
	withDark := plain.WithDark(dark)
	if got := withDark.ForTheme(true); !bytes.Equal(got, dark) {
		t.Error("a dark shell did not get the dark variant")
	}
	if got := withDark.ForTheme(false); !bytes.Equal(got, light) {
		t.Error("a light shell got the dark variant")
	}
}

// WithDark and AsTemplate return copies. An Icon handed to two surfaces must
// not change under the first one because the second asked for a variant.
func TestVariantHelpersDoNotMutateTheOriginal(t *testing.T) {
	original := PNG(testPNG(t))
	original.WithDark([]byte("dark"))
	original.AsTemplate()
	if len(original.Dark) != 0 {
		t.Error("WithDark changed the Icon it was called on")
	}
	if original.Template {
		t.Error("AsTemplate changed the Icon it was called on")
	}
}

func TestFileReadsAPNGAndRejectsWhatIsNot(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.png")
	if err := os.WriteFile(good, testPNG(t), 0o600); err != nil {
		t.Fatal(err)
	}
	ic, err := File(good)
	if err != nil {
		t.Fatalf("File on a real PNG: %v", err)
	}
	if ic.Empty() {
		t.Error("File returned an empty Icon for a real PNG")
	}

	bad := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(bad, []byte("this is not a png"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Catching this here is the point: the alternative is bytes reaching the
	// platform and failing where the cause is no longer visible.
	if _, err := File(bad); err == nil {
		t.Error("File accepted a file that is not a PNG")
	} else if !strings.Contains(err.Error(), "not a PNG") {
		t.Errorf("the error does not say what is wrong: %v", err)
	}

	if _, err := File(filepath.Join(dir, "absent.png")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a missing file should report os.ErrNotExist, got %v", err)
	}
}

func TestImageRoundTripsThroughDecode(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.SetNRGBA(1, 1, color.NRGBA{G: 0xff, A: 0xff})

	ic, err := Image(src)
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	got, err := ic.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Bounds() != src.Bounds() {
		t.Errorf("bounds = %v, want %v", got.Bounds(), src.Bounds())
	}
	r, g, b, a := got.At(1, 1).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Errorf("the pixel did not survive: %d %d %d %d", r, g, b, a)
	}
}

func TestImageAcceptsNilAsNoIcon(t *testing.T) {
	ic, err := Image(nil)
	if err != nil {
		t.Fatalf("Image(nil): %v", err)
	}
	if !ic.Empty() {
		t.Error("Image(nil) produced artwork")
	}
}

func TestDecodeOnAnEmptyIconSaysSo(t *testing.T) {
	if _, err := (Icon{}).Decode(); err == nil {
		t.Error("Decode on an empty Icon reported success")
	}
}

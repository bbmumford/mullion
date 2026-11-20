package shell

import (
	"errors"
	"strings"
	"testing"

	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/bbmumford/mullion/internal/tray"
)

// recorder stands in for the notification area. The conversion from the one
// Item model to the tray's own is the part of this package most likely to be
// wrong, and it cannot be seen on a platform whose backend is not built.
type recorder struct {
	created   bool
	destroyed bool
	visible   bool
	icon      []byte
	tooltip   string
	menu      *tray.Menu
	menuSet   int
	notifies  [][2]string
	ran       bool
	updated   []uint32
	createErr error
	showErr   error
	notifyErr error
}

func (r *recorder) Create() error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = true
	return nil
}
func (r *recorder) SetIcon(png []byte) error     { r.icon = png; return nil }
func (r *recorder) SetTooltip(text string) error { r.tooltip = text; return nil }
func (r *recorder) SetMenu(m *tray.Menu) error   { r.menu = m; r.menuSet++; return nil }
func (r *recorder) ShowNotification(title, message string) error {
	if r.notifyErr != nil {
		return r.notifyErr
	}
	r.notifies = append(r.notifies, [2]string{title, message})
	return nil
}
func (r *recorder) Show() error {
	if r.showErr != nil {
		return r.showErr
	}
	r.visible = true
	return nil
}
func (r *recorder) Hide() error                  { r.visible = false; return nil }
func (r *recorder) Bounds() (int, int, int, int) { return 10, 20, 24, 24 }
func (r *recorder) Run() error                   { r.ran = true; return nil }
func (r *recorder) Destroy()                     { r.destroyed = true }
func (r *recorder) UpdateItem(item *tray.MenuItem) error {
	r.updated = append(r.updated, item.ID())
	return nil
}

// withRecorder points the package's platform seam at a recorder for one test.
func withRecorder(t *testing.T) *recorder {
	t.Helper()
	r := &recorder{}
	saved := newPlatform
	newPlatform = func(*tray.Callbacks) tray.PlatformTray { return r }
	t.Cleanup(func() { newPlatform = saved })
	return r
}

func TestShowInstallsEverythingTheTrayWasGiven(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{
		Icon:    icon.PNG([]byte("icon-bytes")),
		Tooltip: "hover me",
		Name:    "Demo",
		Menu:    Menu{{Label: "Open"}},
	}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if !r.created {
		t.Error("the platform tray was never created")
	}
	if !r.visible {
		t.Error("the tray was created but not shown")
	}
	if string(r.icon) != "icon-bytes" {
		t.Errorf("icon = %q, want %q", r.icon, "icon-bytes")
	}
	if r.tooltip != "hover me" {
		t.Errorf("tooltip = %q, want %q", r.tooltip, "hover me")
	}
	if r.menu == nil || len(r.menu.Items) != 1 {
		t.Fatalf("menu was not installed: %+v", r.menu)
	}
}

// An empty Icon is a supported state: an application may run with no artwork,
// and that must not be reported as a failure or install zero bytes.
func TestShowWithNoIconInstallsNothingAndSucceeds(t *testing.T) {
	r := withRecorder(t)
	if err := (&Tray{Tooltip: "no icon"}).Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if r.icon != nil {
		t.Errorf("an empty Icon still installed %q", r.icon)
	}
}

func TestCreateFailureIsReportedNotSwallowed(t *testing.T) {
	r := withRecorder(t)
	r.createErr = errors.New("no notification area")
	err := (&Tray{}).Show()
	if err == nil {
		t.Fatal("Show reported success although the platform refused to create the tray")
	}
	if !strings.Contains(err.Error(), "no notification area") {
		t.Errorf("the platform's reason was lost: %v", err)
	}
}

// The platform holds a pointer to the Callbacks struct, so a handler assigned
// to the Tray before Show has to reach it. This is the wiring that makes a
// menu-bar application respond to a click at all.
func TestClickHandlersReachThePlatform(t *testing.T) {
	withRecorder(t)
	var left, double, right bool
	tr := &Tray{
		OnClick:       func() { left = true },
		OnDoubleClick: func() { double = true },
		OnRightClick:  func() { right = true },
	}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	tr.impl.Callbacks.OnClick()
	tr.impl.Callbacks.OnDoubleClick()
	tr.impl.Callbacks.OnRightClick()
	if !left || !double || !right {
		t.Errorf("handlers did not arrive: left=%v double=%v right=%v", left, double, right)
	}
}

func TestSetMenuReplacesAndEmptyMenuRemoves(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{Menu: Menu{{Label: "First"}}}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if err := tr.SetMenu(Menu{{Label: "A"}, {Label: "B"}}); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	if r.menu == nil || len(r.menu.Items) != 2 {
		t.Fatalf("the replacement menu did not arrive: %+v", r.menu)
	}
	if err := tr.SetMenu(nil); err != nil {
		t.Fatalf("SetMenu(nil): %v", err)
	}
	if r.menu != nil {
		t.Errorf("an empty Menu left %d items installed", len(r.menu.Items))
	}
}

func TestHideKeepsTheTrayAndShowBringsItBack(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{Icon: icon.PNG([]byte("x"))}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	created := tr.impl
	if err := tr.Hide(); err != nil {
		t.Fatalf("Hide: %v", err)
	}
	if r.visible {
		t.Error("Hide left the tray visible")
	}
	if r.destroyed {
		t.Error("Hide destroyed the tray; Remove is what destroys it")
	}
	if err := tr.Show(); err != nil {
		t.Fatalf("second Show: %v", err)
	}
	if tr.impl != created {
		t.Error("the second Show built a new platform tray instead of reusing the one it had")
	}
}

func TestRemoveDestroysThePlatformTray(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	tr.Remove()
	if !r.destroyed {
		t.Error("Remove did not destroy the platform tray")
	}
}

func TestNotifyBeforeShowSaysSoRatherThanPanicking(t *testing.T) {
	withRecorder(t)
	err := (&Tray{}).Notify("Title", "Body")
	if err == nil {
		t.Fatal("Notify before Show reported success")
	}
	if !strings.Contains(err.Error(), "Show") {
		t.Errorf("the error does not say what to do about it: %v", err)
	}
}

func TestNotifyCarriesTitleAndBody(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if err := Notify(tr, Notification{Title: "Done", Body: "The build finished"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if len(r.notifies) != 1 || r.notifies[0] != [2]string{"Done", "The build finished"} {
		t.Errorf("notification = %v", r.notifies)
	}
}

func TestNotifyWithoutATraySaysWhatToAdd(t *testing.T) {
	err := Notify(nil, Notification{Title: "x"})
	if err == nil {
		t.Fatal("Notify(nil, ...) reported success")
	}
	if !strings.Contains(err.Error(), "Tray") {
		t.Errorf("the error does not name what is missing: %v", err)
	}
}

func TestEmptyNotificationIsRefused(t *testing.T) {
	r := withRecorder(t)
	tr := &Tray{}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	if err := Notify(tr, Notification{}); err == nil {
		t.Fatal("a notification with no title and no body was accepted")
	}
	if len(r.notifies) != 0 {
		t.Errorf("an empty notification still reached the platform: %v", r.notifies)
	}
}

func TestBoundsComeFromThePlatformAndAreZeroBeforeShow(t *testing.T) {
	withRecorder(t)
	tr := &Tray{}
	if x, y, w, h := tr.Bounds(); x|y|w|h != 0 {
		t.Errorf("Bounds before Show = %d,%d %dx%d, want zeroes", x, y, w, h)
	}
	if err := tr.Show(); err != nil {
		t.Fatalf("Show: %v", err)
	}
	x, y, w, h := tr.Bounds()
	if x != 10 || y != 20 || w != 24 || h != 24 {
		t.Errorf("Bounds = %d,%d %dx%d, want 10,20 24x24", x, y, w, h)
	}
}

// Two surfaces driving one event loop is the failure this merge exists to
// prevent, so Run has to refuse rather than start a second loop.
func TestRunRefusesWhenAnotherSurfaceOwnsTheLoop(t *testing.T) {
	r := withRecorder(t)
	if !mainthread.Claim("test-surface") {
		t.Fatalf("the loop was already claimed by %s", mainthread.Owner())
	}
	t.Cleanup(mainthread.Release)

	err := (&Tray{}).Run()
	if err == nil {
		t.Fatal("Run started a second event loop")
	}
	if !strings.Contains(err.Error(), "test-surface") {
		t.Errorf("the error does not name the owner: %v", err)
	}
	if r.ran {
		t.Error("the platform loop was started anyway")
	}
}

func TestRunClaimsTheLoopAndGivesItBack(t *testing.T) {
	r := withRecorder(t)
	if err := (&Tray{}).Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !r.ran {
		t.Error("Run did not start the platform loop")
	}
	if mainthread.Claimed() {
		t.Errorf("Run returned still holding the loop for %s", mainthread.Owner())
	}
}

func TestRunShowsATrayThatWasNotShownYet(t *testing.T) {
	r := withRecorder(t)
	if err := (&Tray{Icon: icon.PNG([]byte("x"))}).Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !r.visible {
		t.Error("Run left the tray invisible")
	}
}

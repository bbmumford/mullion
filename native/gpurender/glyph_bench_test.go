//go:build darwin || linux || android || windows || js

package gpurender

import (
	"strings"
	"testing"
	"time"

	"github.com/bbmumford/mullion/native"
)

// BenchmarkGlyphDrawList times CPU draw-list construction with real shaped text
// and warmed bitmap/atlas caches. Native submission and UI construction are
// outside the timed loop; this isolates the per-repaint renderer work.
func BenchmarkGlyphDrawList(b *testing.B) {
	oldGPU, oldUploads := gpu, pendingUploads
	gpu.init()
	pendingUploads = nil
	defer func() { gpu, pendingUploads = oldGPU, oldUploads }()
	for !native.SystemFontScanDone() {
		time.Sleep(time.Millisecond)
	}
	host := native.GetHost()
	oldSize, oldScale, oldBudget := host.WindowSize, host.WindowScale, host.GlyphCacheBudgetBytes
	defer func() { host.WindowSize, host.WindowScale, host.GlyphCacheBudgetBytes = oldSize, oldScale, oldBudget }()
	host.WindowSize, host.WindowScale, host.GlyphCacheBudgetBytes = native.Vec2{1000, 720}, 2, 32<<20
	for _, mode := range []string{"Uniform", "Mixed", "Legacy", "ClippedLongLines"} {
		b.Run(mode, func(b *testing.B) {
			text := strings.Repeat("func example() { return value } ", 3)
			if mode == "ClippedLongLines" {
				text = strings.Repeat(text, 8)
			}
			var out native.FrameOutputData
			scope := new(int)
			for range 5 {
				out = native.RunFrameFn(func() {
					native.ModAttrs(native.NoAnimate)
					native.ContainerWithKey(scope, native.Attrs(native.Viewport, native.Clip, native.Pad(8)), func() {
						for range 30 {
							native.Container(native.Attrs(native.FixHeight(20), native.Expand, native.Clip), func() {
								native.ModAttrs(native.UnsetMaxCross)
								style := native.TextStyle(native.FontSize(14), native.Fonts(native.Monospace...))
								if mode == "Mixed" || mode == "Legacy" {
									native.Text(text, style, native.Span(0, 4, native.TextColor(40, 90, 40, 1)), native.Span(5, 12, native.TextColor(220, 80, 50, .6)))
								} else {
									native.Text(text, style)
								}
							})
						}
					})
				})
			}
			if mode == "Legacy" {
				var runs []native.GlyphRun
				for i := range out.Surfaces {
					s := &out.Surfaces[i]
					if s.GlyphRunCount == 0 {
						continue
					}
					first := len(runs)
					for j := 0; j < int(s.GlyphRunCount); j++ {
						runs = append(runs, s.GlyphRunAt(j, out.GlyphRuns))
					}
					s.GlyphData = nil
					s.GlyphRunFirst = int32(first)
				}
				out.GlyphRuns = runs
			}
			var build builder
			build.build(out.Surfaces, out.GlyphRuns, 2, 2000, 1440)
			if gpu.failed {
				b.Fatal("fixture exhausted the atlas")
			}
			pendingUploads = nil
			if len(build.quads) < 100 {
				b.Fatalf("fixture only produced %d quads", len(build.quads))
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				build.build(out.Surfaces, out.GlyphRuns, 2, 2000, 1440)
			}
			b.StopTimer()
			b.ReportMetric(float64(len(build.quads)), "quads/frame")
		})
	}
}

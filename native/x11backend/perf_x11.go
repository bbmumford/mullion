//go:build linux || (darwin && x11darwin)

package x11backend

import "github.com/bbmumford/mullion/native"

func perfLog(format string, args ...any) { native.PerfLog(format, args...) }

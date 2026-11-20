//go:build linux

package waylandbackend

import "github.com/bbmumford/mullion/native"

func perfLog(format string, args ...any) { native.PerfLog(format, args...) }

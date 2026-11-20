//go:build !ios && !android

package camera

import "github.com/bbmumford/mullion/native"

func init() {
	takePhoto = func(ctx native.BackendContext, args *TakePhotoArgs) {
		if args != nil && args.Done != nil {
			args.Done(errUnsupported)
		}
	}
}

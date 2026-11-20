//go:build js

package jsbackend

import "github.com/bbmumford/mullion/native"

// Context is the BackendContext for the browser/wasm shell.
type Context struct{}

// Platform implements native.BackendContext.
func (Context) Platform() string { return "js" }

var _ native.BackendContext = Context{}

package main

import "github.com/bbmumford/mullion/internal/brand"

// mullionActivityJava is loaded from the shared embed FS (see ios_embed.go /
// runtimeEmbed). Soft keyboard + InputConnection host; native halves live in
// androidbackend/jni.c.
func mullionActivityJavaBytes() ([]byte, error) {
	return runtimeEmbed.ReadFile("embed/" + brand.AndroidGlueJavaFile())
}

// Package brand holds the names the build tools put into the things they
// produce, in one place rather than spelled out wherever they are used.
//
// The tools came from go-mullion with its name written inline in some two
// hundred places: log prefixes, temporary directory patterns, generated file
// names, environment variables, reverse-DNS identifiers, a logcat tag. Renaming
// a project by substituting a string across two hundred sites is how a tool
// ends up half-renamed, with one forgotten pattern that stops matching the
// directories another part still creates.
//
// So the name lives here. Changing it is one edit, and the places that must
// agree with something outside this module are marked below as the exceptions
// they are.
package brand

import (
	"os"
	"strings"
)

const (
	// Name is the lowercase identifier used in paths, file names, temporary
	// directory patterns and log prefixes.
	Name = "mullion"

	// Title is the name as shown to a person, in a window title or a help
	// banner.
	Title = "Mullion"

	// EnvPrefix begins every environment variable the tools read.
	EnvPrefix = "MULLION"

	// ReverseDNS is the default root for an application identifier, used when
	// a project has not set one of its own.
	ReverseDNS = "dev.mullion"
)

// Android glue, which is the one name that cannot be changed from here alone.
//
// The Java class a mullion Android application runs under declares native
// methods, and JNI binds those to C symbols by a name encoding the Java
// package and class: Java_dev_mullion_host_MullionActivity_nativeCommitText and
// six siblings, written out in native/androidbackend/jni.c. The Java side, the
// manifest, the adb launch component and those C symbols must all say the same
// thing or the application starts and then fails the first time someone types
// into a text field -- at runtime, on a phone, with nothing at build time to
// catch it.
//
// Renaming it therefore means editing jni.c in the same change. These
// constants exist so that edit is two files rather than a hunt, and so the
// coupling is written down where someone renaming things will meet it.
const (
	AndroidGluePackage = "dev.mullion.host"
	AndroidGlueClass   = "MullionActivity"

	// AndroidLogTag is the tag the Android runtime writes logcat lines under,
	// set in native/androidbackend/jni.c. Pinned for the same reason as the
	// class above: the tools print a logcat command for someone to run, and a
	// tag that does not match what the runtime writes shows an empty log,
	// which reads as "the application produced nothing" rather than "you
	// filtered on the wrong word".
	AndroidLogTag = "mullion"
)

// AndroidGlueFQCN is the fully qualified class name, for a manifest or an adb
// component.
func AndroidGlueFQCN() string { return AndroidGluePackage + "." + AndroidGlueClass }

// AndroidGlueJavaFile is the source file name the glue class must live in,
// which Java requires to match the class.
func AndroidGlueJavaFile() string { return AndroidGlueClass + ".java" }

// AndroidLibName is the shared library an Android build produces and the glue
// class loads: lib<name>.so, named in the manifest's android.app.lib_name and
// in System.loadLibrary. Tool-side from end to end, so unlike the glue class
// this one moves freely -- but all four places must still agree.
const AndroidLibName = Name + "app"

// CapabilitiesFile is the file a package ships to declare the platform
// permissions it needs, which the mobile tools union across a build's
// dependencies.
const CapabilitiesFile = Name + ".capabilities"

// Env reads an environment variable by its unprefixed name: Env("IOS_TEAM")
// reads MULLION_IOS_TEAM.
func Env(name string) string {
	return os.Getenv(EnvPrefix + "_" + name)
}

// EnvName is the variable a message should name when telling someone to set
// one.
func EnvName(name string) string { return EnvPrefix + "_" + name }

// LogPrefix is what a tool puts in front of its messages: "mullion-bundle: ".
func LogPrefix(tool string) string { return Name + "-" + tool + ": " }

// TempPattern is a pattern for a temporary directory belonging to a tool, for
// os.MkdirTemp: "mullion-bundle-*".
func TempPattern(tool string) string { return Name + "-" + tool + "-*" }

// AppID builds a reverse-DNS application identifier from a bare name, leaving
// one that already looks qualified alone.
func AppID(name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	return ReverseDNS + "." + strings.ToLower(name)
}

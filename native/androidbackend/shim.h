// C side of the mullion Android backend. android_main (called by the vendored
// native_app_glue on the app thread) forwards lifecycle commands and touch
// input to Go exports, then hands control to Go via mullion_android_main.
#ifndef MULLION_ANDROID_SHIM_H
#define MULLION_ANDROID_SHIM_H

#include <stdint.h>

// Flags returned by mullion_android_poll.
#define MULLION_POLL_DESTROY 1
#define MULLION_POLL_HAS_WINDOW 2

// mullion_android_poll blocks up to timeoutMillis (-1 = forever) for looper
// activity, drains every pending source (glue commands + input events —
// callbacks into Go fire from inside), and reports current state flags.
int mullion_android_poll(int timeoutMillis);

// Window geometry in device pixels; 0 when there is no window.
int mullion_android_window_width(void);
int mullion_android_window_height(void);

// Current ANativeWindow* (NULL if none). For EGL; do not lock this window
// while an EGL surface owns it.
void *mullion_android_native_window(void);

// Set the window to CPU-lockable RGBA8888. Call before the software present
// path; skip when EGL is presenting.
void mullion_android_use_cpu_buffers(void);

// The visible content rect within the window (excludes status/nav bars and,
// with adjustResize, the soft keyboard — the surface itself never resizes;
// only this rect does). All zeros when unknown.
void mullion_android_content_rect(int* l, int* t, int* r, int* b);

// mullion_android_lock_window locks the front buffer for CPU rendering.
// Returns 1 on success and fills bits/w/h/stridePx (stride in PIXELS, as
// ANativeWindow_Buffer reports it). Pair with mullion_android_unlock_post.
int mullion_android_lock_window(void** bits, int* w, int* h, int* stridePx);
void mullion_android_unlock_post(void);

// Screen density in dpi (160 = mdpi baseline); 0 if unknown.
float mullion_android_density(void);

// One line to logcat under the "mullion" tag (stdout/stderr redirect).
void mullion_android_log(const char* msg);

// mullion_android_app exposes the glue state to jni.c.
struct android_app;
struct android_app* mullion_android_app(void);

// mullion_android_wake breaks the app thread out of a blocked poll (called
// from UI-thread JNI natives after queueing input for the next frame).
void mullion_android_wake(void);

// Soft keyboard show/hide via the Java activity (see jni.c). App thread only.
void mullion_android_set_keyboard(int show);

// Preferred orientation (matches native.PreferredOrientation): 0 = any,
// 1 = portrait, 2 = landscape. App thread only; hops to the UI thread.
void mullion_android_set_orientation(int preferred);

// 1 when a physical keyboard is available (config QWERTY/12KEY or an
// alphabetic non-virtual InputDevice), 0 for soft-IME-only. App thread.
int mullion_android_hardware_keyboard(void);

// Keycode+meta → unicode via KeyCharacterMap in Java (see jni.c). 0 = none.
int mullion_android_key_unicode(int deviceId, int keyCode, int metaState);

// Clipboard via the Java activity (see jni.c). App thread only. The returned
// string is malloc'd — caller frees; NULL when empty/unavailable.
void mullion_android_set_clipboard(const char* text);
char* mullion_android_get_clipboard(void);

// Open url in the system browser / scheme handler (Intent.ACTION_VIEW).
// App thread only; hops to the UI thread inside the activity.
void mullion_android_open_url(const char* url);

// ---- generic extension host (no feature-specific logic) --------------------

// Current MullionActivity jobject (glue-owned; do not DeleteLocalRef).
// NULL if the activity is not available.
void *mullion_android_activity(void);

// JNIEnv* for the current (app) thread after AttachCurrentThread; NULL on fail.
// Shared with extension C code that needs to build Intents, etc.
void *mullion_android_jni_env(void);

// Start an activity for result. intent is a local or global jobject Intent.
// Safe from the app thread; hops to the UI thread. Result is delivered via
// mullionAndroidActivityResult (Go) after onActivityResult.
void mullion_android_start_activity_for_result(void *intent /* jobject */, int request_code);

// Request runtime permissions. perms is a NULL-terminated list of UTF-8 names
// (e.g. "android.permission.CAMERA"). Result via mullionAndroidPermissionResult.
void mullion_android_request_permissions(const char **perms, int request_code);

// checkSelfPermission: 0 = granted, non-zero = denied/unavailable.
int mullion_android_check_permission(const char *perm);

#endif

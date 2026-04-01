// MobileRUM — Real User Monitoring for Android
//
// Captures mobile-specific performance and interaction metrics:
//   - App launch time (cold start, warm start, hot start)
//   - Activity/Fragment transitions with timing
//   - Touch interaction latency (tap to response)
//   - Memory and CPU usage snapshots
//   - Network connectivity changes (WiFi, cellular, offline)
//   - ANR (Application Not Responding) detection
//   - Custom user sessions with automatic timeout
//
// Data is batched and sent as OTel spans/events to the configured endpoint.
// Sampling is controlled by CloudMockConfig.sampleRate.
//
// Usage:
//   // Automatic — enabled via CloudMock.initialize() when enableRUM = true
//   // Manual screen tracking:
//   MobileRUM.trackScreen("HomeActivity")
//   MobileRUM.trackInteraction("tap", "checkout_button")
package io.cloudmock

/**
 * MobileRUM provides real user monitoring for Android applications.
 */
object MobileRUM {

    /** Start a new RUM session. Called automatically by CloudMock.initialize(). */
    fun startSession() {
        // TODO: Generate session ID, register ActivityLifecycleCallbacks,
        // begin recording lifecycle events.
    }

    /** Track a screen view transition. Call from onResume or equivalent. */
    fun trackScreen(name: String, attributes: Map<String, String> = emptyMap()) {
        // TODO: Record screen view as an OTel span with timing.
    }

    /** Track a user interaction (tap, swipe, scroll, etc). */
    fun trackInteraction(type: String, target: String, attributes: Map<String, String> = emptyMap()) {
        // TODO: Record interaction as an OTel event.
    }

    /** Record app launch timing. Call from Application.onCreate(). */
    fun recordAppLaunch(coldStart: Boolean, durationMs: Double) {
        // TODO: Emit app launch metric.
    }

    /** Record a memory/CPU snapshot. Called periodically by the SDK. */
    fun recordResourceSnapshot(memoryMB: Double, cpuPercent: Double) {
        // TODO: Emit resource usage metric.
    }

    /** End the current session. Called on app background or explicit logout. */
    fun endSession() {
        // TODO: Flush session data, mark session as ended.
    }
}

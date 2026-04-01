// MobileRUM — Real User Monitoring for iOS/macOS
//
// Captures mobile-specific performance and interaction metrics:
//   - App launch time (cold start, warm start, hot start)
//   - Screen/view transitions with timing
//   - Touch interaction latency (tap → response)
//   - Memory and CPU usage snapshots
//   - Network connectivity changes (WiFi, cellular, offline)
//   - Crash and ANR (Application Not Responding) detection
//   - Custom user sessions with automatic timeout
//
// Data is batched and sent as OTel spans/events to the configured endpoint.
// Sampling is controlled by CloudMockConfig.sampleRate.
//
// Usage:
//   // Automatic — enabled via CloudMock.initialize(config:) when enableRUM = true
//   // Manual screen tracking:
//   MobileRUM.trackScreen("HomeViewController")
//   MobileRUM.trackInteraction("tap", target: "checkout_button")

import Foundation

/// MobileRUM provides real user monitoring for mobile applications.
public class MobileRUM {

    /// Start a new RUM session. Called automatically by CloudMock.initialize.
    public static func startSession() {
        // TODO: Generate session ID, begin recording lifecycle events.
    }

    /// Track a screen view transition. Call from viewDidAppear or equivalent.
    public static func trackScreen(_ name: String, attributes: [String: String] = [:]) {
        // TODO: Record screen view as an OTel span with timing.
    }

    /// Track a user interaction (tap, swipe, scroll, etc).
    public static func trackInteraction(_ type: String, target: String, attributes: [String: String] = [:]) {
        // TODO: Record interaction as an OTel event.
    }

    /// Record app launch timing. Call from application:didFinishLaunching.
    public static func recordAppLaunch(coldStart: Bool, durationMs: Double) {
        // TODO: Emit app launch metric.
    }

    /// Record a memory/CPU snapshot. Called periodically by the SDK.
    public static func recordResourceSnapshot(memoryMB: Double, cpuPercent: Double) {
        // TODO: Emit resource usage metric.
    }

    /// End the current session. Called on app background or explicit logout.
    public static func endSession() {
        // TODO: Flush session data, mark session as ended.
    }
}

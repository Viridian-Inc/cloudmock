// CloudMock Swift SDK — thin OTel wrapper for iOS/macOS
// Provides: URLSession interception, mobile RUM, BLE mesh topology

import Foundation

public struct CloudMockConfig {
    public var endpoint: String
    public var appName: String
    public var enableRUM: Bool
    public var enableBLE: Bool
    public var sampleRate: Double

    public init(
        endpoint: String = "http://localhost:4318",
        appName: String = "app",
        enableRUM: Bool = true,
        enableBLE: Bool = false,
        sampleRate: Double = 1.0
    ) {
        self.endpoint = endpoint
        self.appName = appName
        self.enableRUM = enableRUM
        self.enableBLE = enableBLE
        self.sampleRate = sampleRate
    }
}

/// CloudMock is the main entry point for the iOS/macOS SDK.
/// It initializes telemetry collection and provides a simple API for
/// tracking custom events, capturing errors, and flushing buffered data.
public class CloudMock {
    private static var config: CloudMockConfig?
    private static var isInitialized = false

    /// Initialize the CloudMock SDK with the given configuration.
    /// Must be called before any other SDK methods.
    /// Typically called in `application(_:didFinishLaunchingWithOptions:)`.
    public static func initialize(config: CloudMockConfig) {
        self.config = config
        self.isInitialized = true
        // TODO: Set up OTel TracerProvider, wire URLSession interceptor,
        // start RUM session if enabled, start BLE mesh if enabled.
    }

    /// Track a named event with optional string attributes.
    /// Events are buffered and sent in batches to the configured endpoint.
    public static func track(name: String, attributes: [String: String] = [:]) {
        guard isInitialized else { return }
        // TODO: Create an OTel span or event with the given name and attributes.
    }

    /// Capture an error and attach it to the current trace context.
    /// The error's localizedDescription and stack trace are recorded.
    public static func captureError(_ error: Error) {
        guard isInitialized else { return }
        // TODO: Record error as an OTel span event with error attributes.
    }

    /// Flush any buffered telemetry data to the endpoint immediately.
    /// Call this before app termination or backgrounding.
    public static func flush() {
        guard isInitialized else { return }
        // TODO: Force-flush the OTel span exporter.
    }
}

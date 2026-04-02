import Foundation

/// Main entry point for the CloudMock devtools SDK.
///
/// Usage:
/// ```swift
/// CloudMock.shared.start(appName: "my-ios-app")
/// ```
public final class CloudMock {
    public static let shared = CloudMock()

    private(set) var connection: Connection?
    private var urlSessionInterceptor: URLSessionInterceptor?
    private var errorInterceptor: ErrorInterceptor?
    private var isStarted = false

    private init() {}

    /// Start the CloudMock SDK and connect to devtools.
    /// No-ops if devtools isn't running.
    ///
    /// - Parameters:
    ///   - appName: Name shown in the devtools source bar
    ///   - host: Devtools host (default: localhost)
    ///   - port: Devtools TCP port (default: 4580)
    public func start(appName: String, host: String = "localhost", port: Int = 4580) {
        guard !isStarted else { return }
        isStarted = true

        let conn = Connection(host: host, port: port, appName: appName)
        self.connection = conn

        // Register this source
        conn.send(event: SourceEvent(
            type: "source:register",
            data: [
                "runtime": "swift",
                "appName": appName,
                "pid": ProcessInfo.processInfo.processIdentifier
            ],
            source: appName,
            runtime: "swift"
        ))

        // Set up interceptors
        urlSessionInterceptor = URLSessionInterceptor(connection: conn)
        urlSessionInterceptor?.install()

        errorInterceptor = ErrorInterceptor(connection: conn)
        errorInterceptor?.install()
    }

    /// Log a message to devtools.
    ///
    /// - Parameters:
    ///   - message: The log message
    ///   - level: Log level (default: .info)
    ///   - file: Source file (auto-populated)
    ///   - line: Source line (auto-populated)
    public func log(
        _ message: String,
        level: LogLevel = .info,
        file: String = #file,
        line: Int = #line
    ) {
        connection?.send(event: SourceEvent(
            type: "console",
            data: [
                "level": level.rawValue,
                "message": message,
                "file": file,
                "line": line
            ],
            source: connection?.appName ?? "swift-app",
            runtime: "swift"
        ))
    }

    /// Stop the SDK and disconnect from devtools.
    public func stop() {
        guard isStarted else { return }

        urlSessionInterceptor?.uninstall()
        urlSessionInterceptor = nil

        errorInterceptor?.uninstall()
        errorInterceptor = nil

        connection?.close()
        connection = nil
        isStarted = false
    }

    deinit {
        stop()
    }
}

// MARK: - Log Level

public enum LogLevel: String {
    case debug
    case info
    case warn
    case error
}

// MARK: - Source Event

struct SourceEvent: Encodable {
    let type: String
    let data: [String: Any]
    let source: String
    let runtime: String
    let timestamp: UInt64

    init(type: String, data: [String: Any], source: String, runtime: String) {
        self.type = type
        self.data = data
        self.source = source
        self.runtime = runtime
        self.timestamp = UInt64(Date().timeIntervalSince1970 * 1000)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(type, forKey: .type)
        try container.encode(source, forKey: .source)
        try container.encode(runtime, forKey: .runtime)
        try container.encode(timestamp, forKey: .timestamp)

        // Encode heterogeneous data dictionary
        let jsonData = try JSONSerialization.data(withJSONObject: data)
        let jsonObject = try JSONSerialization.jsonObject(with: jsonData)
        let dataWrapper = AnyCodable(jsonObject)
        try container.encode(dataWrapper, forKey: .data)
    }

    enum CodingKeys: String, CodingKey {
        case type, data, source, runtime, timestamp
    }
}

// MARK: - AnyCodable wrapper for heterogeneous dictionaries

struct AnyCodable: Encodable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()

        switch value {
        case let val as String:
            try container.encode(val)
        case let val as Int:
            try container.encode(val)
        case let val as Int32:
            try container.encode(val)
        case let val as UInt64:
            try container.encode(val)
        case let val as Double:
            try container.encode(val)
        case let val as Bool:
            try container.encode(val)
        case let val as [String: Any]:
            let dict = val.mapValues { AnyCodable($0) }
            try container.encode(dict)
        case let val as [Any]:
            let arr = val.map { AnyCodable($0) }
            try container.encode(arr)
        case is NSNull:
            try container.encodeNil()
        default:
            try container.encode(String(describing: value))
        }
    }
}

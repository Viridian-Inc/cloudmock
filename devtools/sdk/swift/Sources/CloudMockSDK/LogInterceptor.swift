import Foundation

/// Provides structured logging that forwards to CloudMock devtools.
///
/// Usage:
/// ```swift
/// CloudMock.shared.log("User tapped button", level: .info)
/// CloudMock.shared.log("Network timeout", level: .error)
/// ```
///
/// This file also installs an NSSetUncaughtExceptionHandler bridge
/// that captures NSLog output when possible. Since NSLog interception
/// is limited on modern Apple platforms, the primary API is CloudMock.log().
///
/// For complete logging, use `CloudMock.shared.log()` directly instead of NSLog/print.

// MARK: - Log capture via redirecting stderr

/// Captures stderr output (which includes NSLog) by redirecting the file descriptor.
/// This is a best-effort approach; not all NSLog output may be captured.
final class LogInterceptor {
    private let connection: Connection
    private var isInstalled = false
    private var originalStderr: Int32 = -1
    private var pipe: Pipe?
    private var readSource: DispatchSourceRead?

    init(connection: Connection) {
        self.connection = connection
    }

    func install() {
        guard !isInstalled else { return }
        isInstalled = true

        // Redirect stderr through a pipe to capture NSLog output
        let pipe = Pipe()
        self.pipe = pipe

        originalStderr = dup(STDERR_FILENO)
        dup2(pipe.fileHandleForWriting.fileDescriptor, STDERR_FILENO)

        let readFD = pipe.fileHandleForReading.fileDescriptor
        let source = DispatchSource.makeReadSource(fileDescriptor: readFD, queue: .global(qos: .utility))
        self.readSource = source

        source.setEventHandler { [weak self] in
            guard let self else { return }
            let data = pipe.fileHandleForReading.availableData
            guard !data.isEmpty else { return }

            // Write to original stderr so logs still appear
            if self.originalStderr >= 0 {
                data.withUnsafeBytes { bytes in
                    if let ptr = bytes.baseAddress {
                        write(self.originalStderr, ptr, data.count)
                    }
                }
            }

            // Forward to devtools
            if let message = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines),
               !message.isEmpty {
                self.connection.send(event: SourceEvent(
                    type: "console",
                    data: [
                        "level": "info",
                        "message": message,
                    ],
                    source: self.connection.appName,
                    runtime: "swift"
                ))
            }
        }

        source.resume()
    }

    func uninstall() {
        guard isInstalled else { return }
        isInstalled = false

        readSource?.cancel()
        readSource = nil

        // Restore original stderr
        if originalStderr >= 0 {
            dup2(originalStderr, STDERR_FILENO)
            close(originalStderr)
            originalStderr = -1
        }

        pipe = nil
    }
}

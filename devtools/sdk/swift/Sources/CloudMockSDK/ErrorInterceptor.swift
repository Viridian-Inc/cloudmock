import Foundation

/// Captures uncaught NSExceptions and sends them to devtools.
/// Uses NSSetUncaughtExceptionHandler to intercept ObjC exceptions.
/// Swift-native errors (fatalError, etc.) are not catchable at runtime.
final class ErrorInterceptor {
    private let connection: Connection
    private var isInstalled = false
    private static weak var activeConnection: Connection?
    private static var previousHandler: (@convention(c) (NSException) -> Void)?

    init(connection: Connection) {
        self.connection = connection
    }

    func install() {
        guard !isInstalled else { return }
        isInstalled = true

        ErrorInterceptor.activeConnection = connection

        // Save previous handler and install ours
        ErrorInterceptor.previousHandler = NSGetUncaughtExceptionHandler()
        NSSetUncaughtExceptionHandler { exception in
            ErrorInterceptor.handleException(exception)
        }

        // Also install signal handlers for common crash signals
        installSignalHandlers()
    }

    func uninstall() {
        guard isInstalled else { return }
        isInstalled = false

        // Restore previous handler
        if let previous = ErrorInterceptor.previousHandler {
            NSSetUncaughtExceptionHandler(previous)
        } else {
            NSSetUncaughtExceptionHandler(nil)
        }

        ErrorInterceptor.activeConnection = nil
        ErrorInterceptor.previousHandler = nil
    }

    // MARK: - Exception handling

    private static func handleException(_ exception: NSException) {
        guard let conn = activeConnection else { return }

        let symbols = exception.callStackSymbols
        let stack = symbols.joined(separator: "\n")

        conn.send(event: SourceEvent(
            type: "error:uncaught",
            data: [
                "name": exception.name.rawValue,
                "message": exception.reason ?? "Unknown exception",
                "stack": stack,
                "userInfo": String(describing: exception.userInfo ?? [:])
            ],
            source: conn.appName,
            runtime: "swift"
        ))

        // Chain to previous handler
        previousHandler?(exception)
    }

    // MARK: - Signal handlers

    private func installSignalHandlers() {
        let signals: [Int32] = [SIGABRT, SIGBUS, SIGFPE, SIGILL, SIGSEGV, SIGTRAP]

        for sig in signals {
            signal(sig) { signalNumber in
                guard let conn = ErrorInterceptor.activeConnection else { return }

                let signalName: String
                switch signalNumber {
                case SIGABRT: signalName = "SIGABRT"
                case SIGBUS: signalName = "SIGBUS"
                case SIGFPE: signalName = "SIGFPE"
                case SIGILL: signalName = "SIGILL"
                case SIGSEGV: signalName = "SIGSEGV"
                case SIGTRAP: signalName = "SIGTRAP"
                default: signalName = "SIGNAL(\(signalNumber))"
                }

                conn.send(event: SourceEvent(
                    type: "error:uncaught",
                    data: [
                        "name": signalName,
                        "message": "Process received \(signalName)",
                        "stack": Thread.callStackSymbols.joined(separator: "\n")
                    ],
                    source: conn.appName,
                    runtime: "swift"
                ))

                // Re-raise with default handler
                signal(signalNumber, SIG_DFL)
                raise(signalNumber)
            }
        }
    }
}

import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif

/// CloudMockServer starts the CloudMock binary and exposes a local AWS-compatible endpoint.
/// Use this in unit/integration tests instead of the mobile RUM `CloudMock` entry point.
public class CloudMockServer {
    public let port: Int
    public var endpoint: String { "http://localhost:\(port)" }

    private var process: Process?

    public init(port: Int = 0, profile: String = "minimal") {
        self.port = port > 0 ? port : Self.findFreePort()
    }

    public func start() throws {
        let proc = Process()
        proc.executableURL = URL(fileURLWithPath: "/usr/bin/env")
        proc.arguments = ["cloudmock", "--port", "\(port)"]
        proc.environment = ProcessInfo.processInfo.environment.merging([
            "CLOUDMOCK_PROFILE": "minimal",
            "CLOUDMOCK_IAM_MODE": "none"
        ]) { _, new in new }
        proc.standardOutput = FileHandle.nullDevice
        proc.standardError = FileHandle.nullDevice

        try proc.run()
        self.process = proc
        try waitForReady(timeout: 30)
    }

    public func stop() {
        process?.terminate()
        process?.waitUntilExit()
        process = nil
    }

    deinit { stop() }

    private func waitForReady(timeout: TimeInterval) throws {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            let fd = socket(AF_INET, SOCK_STREAM, 0)
            guard fd >= 0 else { Thread.sleep(forTimeInterval: 0.1); continue }
            defer { close(fd) }

            var addr = sockaddr_in()
            addr.sin_family = sa_family_t(AF_INET)
            addr.sin_port = UInt16(port).bigEndian
            addr.sin_addr.s_addr = inet_addr("127.0.0.1")

            let result = withUnsafePointer(to: &addr) { ptr in
                ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockPtr in
                    Darwin.connect(fd, sockPtr, socklen_t(MemoryLayout<sockaddr_in>.size))
                }
            }
            if result == 0 { return }
            Thread.sleep(forTimeInterval: 0.1)
        }
        throw NSError(domain: "CloudMock", code: 1, userInfo: [NSLocalizedDescriptionKey: "Timeout waiting for CloudMock to start"])
    }

    private static func findFreePort() -> Int {
        let fd = socket(AF_INET, SOCK_STREAM, 0)
        guard fd >= 0 else { return 4566 }
        defer { close(fd) }

        var addr = sockaddr_in()
        addr.sin_family = sa_family_t(AF_INET)
        addr.sin_port = 0
        addr.sin_addr.s_addr = inet_addr("127.0.0.1")

        _ = withUnsafePointer(to: &addr) { ptr in
            ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockPtr in
                bind(fd, sockPtr, socklen_t(MemoryLayout<sockaddr_in>.size))
            }
        }

        var bound = sockaddr_in()
        var len = socklen_t(MemoryLayout<sockaddr_in>.size)
        _ = withUnsafeMutablePointer(to: &bound) { ptr in
            ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockPtr in
                getsockname(fd, sockPtr, &len)
            }
        }
        return Int(UInt16(bigEndian: bound.sin_port))
    }
}

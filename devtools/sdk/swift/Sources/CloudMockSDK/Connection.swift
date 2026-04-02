import Foundation
import Network

/// TCP JSON-line client using Network.framework (NWConnection).
/// Connects to the devtools source server and sends events as newline-delimited JSON.
/// Auto-reconnects on failure and buffers messages when disconnected.
final class Connection {
    let appName: String
    private let host: String
    private let port: Int
    private var nwConnection: NWConnection?
    private let queue = DispatchQueue(label: "io.cloudmock.connection", qos: .utility)
    private var buffer: [Data] = []
    private let maxBufferSize = 100
    private var isConnected = false
    private var isClosed = false
    private var reconnectWorkItem: DispatchWorkItem?

    init(host: String, port: Int, appName: String) {
        self.host = host
        self.port = port
        self.appName = appName
        connect()
    }

    // MARK: - Connection lifecycle

    private func connect() {
        guard !isClosed else { return }

        let nwHost = NWEndpoint.Host(host)
        let nwPort = NWEndpoint.Port(integerLiteral: UInt16(port))

        let connection = NWConnection(host: nwHost, port: nwPort, using: .tcp)
        self.nwConnection = connection

        connection.stateUpdateHandler = { [weak self] state in
            guard let self else { return }
            switch state {
            case .ready:
                self.isConnected = true
                self.flushBuffer()
            case .failed, .cancelled:
                self.isConnected = false
                self.nwConnection = nil
                self.scheduleReconnect()
            case .waiting:
                // Network path not available; will auto-retry or we reconnect
                self.isConnected = false
                self.nwConnection?.cancel()
            default:
                break
            }
        }

        connection.start(queue: queue)
    }

    private func scheduleReconnect() {
        guard !isClosed, reconnectWorkItem == nil else { return }

        let workItem = DispatchWorkItem { [weak self] in
            self?.reconnectWorkItem = nil
            self?.connect()
        }
        reconnectWorkItem = workItem
        queue.asyncAfter(deadline: .now() + 5.0, execute: workItem)
    }

    // MARK: - Sending

    func send(event: SourceEvent) {
        queue.async { [weak self] in
            self?.sendOnQueue(event: event)
        }
    }

    private func sendOnQueue(event: SourceEvent) {
        guard let jsonData = try? JSONEncoder().encode(event) else { return }

        // Append newline to make it JSON-per-line
        var lineData = jsonData
        lineData.append(contentsOf: [0x0A]) // '\n'

        if isConnected, let connection = nwConnection {
            connection.send(content: lineData, completion: .contentProcessed { [weak self] error in
                if error != nil {
                    // Send failed; buffer for retry
                    self?.bufferMessage(lineData)
                }
            })
        } else {
            bufferMessage(lineData)
        }
    }

    private func bufferMessage(_ data: Data) {
        if buffer.count < maxBufferSize {
            buffer.append(data)
        }
    }

    private func flushBuffer() {
        guard isConnected, let connection = nwConnection else { return }

        let pending = buffer
        buffer.removeAll()

        for data in pending {
            connection.send(content: data, completion: .contentProcessed { _ in })
        }
    }

    // MARK: - Cleanup

    func close() {
        isClosed = true
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        nwConnection?.cancel()
        nwConnection = nil
        isConnected = false
        buffer.removeAll()
    }

    deinit {
        close()
    }
}

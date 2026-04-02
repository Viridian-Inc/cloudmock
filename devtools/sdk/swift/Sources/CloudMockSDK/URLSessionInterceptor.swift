import Foundation

/// Intercepts URLSession HTTP traffic by swizzling dataTask methods.
/// Captures request and response details and sends them to devtools.
final class URLSessionInterceptor {
    private let connection: Connection
    private var isInstalled = false

    // Track whether swizzling has been applied globally (across instances)
    private static var swizzled = false
    fileprivate static weak var activeConnection: Connection?

    fileprivate static var requestCounter = 0

    init(connection: Connection) {
        self.connection = connection
    }

    func install() {
        guard !isInstalled else { return }
        isInstalled = true

        URLSessionInterceptor.activeConnection = connection

        guard !URLSessionInterceptor.swizzled else { return }
        URLSessionInterceptor.swizzled = true

        // Register our custom URLProtocol to intercept all URLSession traffic
        URLProtocol.registerClass(CloudMockURLProtocol.self)

        // Also add to default session configuration
        let defaultConfig = URLSessionConfiguration.default
        var protocols = defaultConfig.protocolClasses ?? []
        protocols.insert(CloudMockURLProtocol.self, at: 0)
        defaultConfig.protocolClasses = protocols
    }

    func uninstall() {
        guard isInstalled else { return }
        isInstalled = false
        URLSessionInterceptor.activeConnection = nil
        URLProtocol.unregisterClass(CloudMockURLProtocol.self)
        URLSessionInterceptor.swizzled = false
    }

    // MARK: - Internal access for URLProtocol

    static func sendHTTPEvent(
        id: String,
        request: URLRequest,
        response: HTTPURLResponse?,
        body: Data?,
        duration: TimeInterval,
        error: Error?
    ) {
        guard let conn = activeConnection else { return }

        let url = request.url?.absoluteString ?? "unknown"
        let method = request.httpMethod ?? "GET"
        let path = request.url?.path ?? "/"

        var requestHeaders: [String: String] = [:]
        request.allHTTPHeaderFields?.forEach { key, value in
            requestHeaders[key] = value
        }

        if let error = error {
            conn.send(event: SourceEvent(
                type: "http:error",
                data: [
                    "id": id,
                    "method": method,
                    "url": url,
                    "path": path,
                    "error": error.localizedDescription,
                    "duration_ms": Int(duration * 1000)
                ],
                source: conn.appName,
                runtime: "swift"
            ))
            return
        }

        var responseHeaders: [String: String] = [:]
        response?.allHeaderFields.forEach { key, value in
            responseHeaders[String(describing: key)] = String(describing: value)
        }

        let bodyStr: String
        if let body, !body.isEmpty {
            let truncated = body.prefix(4096)
            bodyStr = String(data: truncated, encoding: .utf8) ?? "<binary \(body.count) bytes>"
        } else {
            bodyStr = ""
        }

        conn.send(event: SourceEvent(
            type: "http:response",
            data: [
                "id": id,
                "method": method,
                "url": url,
                "path": path,
                "status": response?.statusCode ?? 0,
                "duration_ms": Int(duration * 1000),
                "request_headers": requestHeaders,
                "response_headers": responseHeaders,
                "response_body": bodyStr,
                "content_length": responseHeaders["Content-Length"] ?? ""
            ],
            source: conn.appName,
            runtime: "swift"
        ))
    }

    static func nextRequestId() -> String {
        requestCounter += 1
        return "req_\(requestCounter)_\(Int(Date().timeIntervalSince1970 * 1000))"
    }
}

// MARK: - Custom URLProtocol for interception

/// URLProtocol subclass that intercepts HTTP(S) requests, injects headers,
/// captures response data, and forwards everything to the original destination.
final class CloudMockURLProtocol: URLProtocol {
    private static let handledKey = "X-CloudMock-Handled"
    private var dataTask: URLSessionDataTask?
    private var receivedData = Data()
    private var receivedResponse: HTTPURLResponse?
    private var startTime = Date()
    private var requestId = ""

    private lazy var internalSession: URLSession = {
        let config = URLSessionConfiguration.default
        // Remove our protocol from the internal session to avoid recursion
        config.protocolClasses = config.protocolClasses?.filter { $0 !== CloudMockURLProtocol.self }
        return URLSession(configuration: config, delegate: InternalDelegate(parent: self), delegateQueue: nil)
    }()

    override class func canInit(with request: URLRequest) -> Bool {
        // Only intercept HTTP/HTTPS requests we haven't handled yet
        guard let scheme = request.url?.scheme, ["http", "https"].contains(scheme) else {
            return false
        }
        return URLProtocol.property(forKey: handledKey, in: request) == nil
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        return request
    }

    override func startLoading() {
        requestId = URLSessionInterceptor.nextRequestId()
        startTime = Date()

        // Mark this request so we don't intercept it again
        let mutableRequest = (request as NSURLRequest).mutableCopy() as! NSMutableURLRequest
        URLProtocol.setProperty(true, forKey: CloudMockURLProtocol.handledKey, in: mutableRequest)

        // Inject correlation headers
        mutableRequest.setValue(URLSessionInterceptor.activeConnection?.appName ?? "swift-app",
                              forHTTPHeaderField: "X-CloudMock-Source")
        mutableRequest.setValue(requestId, forHTTPHeaderField: "X-CloudMock-Request-Id")

        dataTask = internalSession.dataTask(with: mutableRequest as URLRequest)
        dataTask?.resume()
    }

    override func stopLoading() {
        dataTask?.cancel()
        dataTask = nil
    }

    // MARK: - Internal delegate to forward data back to the URLProtocol client

    fileprivate func didReceive(response: URLResponse) {
        receivedResponse = response as? HTTPURLResponse
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
    }

    fileprivate func didLoad(data: Data) {
        receivedData.append(data)
        client?.urlProtocol(self, didLoad: data)
    }

    fileprivate func didFinish() {
        let duration = Date().timeIntervalSince(startTime)

        URLSessionInterceptor.sendHTTPEvent(
            id: requestId,
            request: request,
            response: receivedResponse,
            body: receivedData,
            duration: duration,
            error: nil
        )

        client?.urlProtocolDidFinishLoading(self)
    }

    fileprivate func didFail(with error: Error) {
        let duration = Date().timeIntervalSince(startTime)

        URLSessionInterceptor.sendHTTPEvent(
            id: requestId,
            request: request,
            response: nil,
            body: nil,
            duration: duration,
            error: error
        )

        client?.urlProtocol(self, didFailWithError: error)
    }
}

// MARK: - Internal URLSession delegate

private final class InternalDelegate: NSObject, URLSessionDataDelegate {
    weak var parent: CloudMockURLProtocol?

    init(parent: CloudMockURLProtocol) {
        self.parent = parent
    }

    func urlSession(_ session: URLSession, dataTask: URLSessionDataTask, didReceive response: URLResponse,
                    completionHandler: @escaping (URLSession.ResponseDisposition) -> Void) {
        parent?.didReceive(response: response)
        completionHandler(.allow)
    }

    func urlSession(_ session: URLSession, dataTask: URLSessionDataTask, didReceive data: Data) {
        parent?.didLoad(data: data)
    }

    func urlSession(_ session: URLSession, task: URLSessionTask, didCompleteWithError error: Error?) {
        if let error {
            parent?.didFail(with: error)
        } else {
            parent?.didFinish()
        }
    }
}

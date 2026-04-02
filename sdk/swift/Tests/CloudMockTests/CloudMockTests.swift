import XCTest
@testable import CloudMock

final class CloudMockTests: XCTestCase {
    func testEndpointFormat() {
        let cm = CloudMockServer(port: 4566)
        XCTAssertEqual(cm.endpoint, "http://localhost:4566")
    }

    func testAutoPort() {
        let cm = CloudMockServer()
        XCTAssertGreaterThan(cm.port, 0)
    }
}

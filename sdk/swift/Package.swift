// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "CloudMock",
    products: [.library(name: "CloudMock", targets: ["CloudMock"])],
    targets: [
        .target(name: "CloudMock"),
        .testTarget(name: "CloudMockTests", dependencies: ["CloudMock"]),
    ]
)

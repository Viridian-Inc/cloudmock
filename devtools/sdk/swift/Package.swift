// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "CloudMockSDK",
    platforms: [
        .macOS(.v13),
        .iOS(.v16),
    ],
    products: [
        .library(
            name: "CloudMockSDK",
            targets: ["CloudMockSDK"]
        ),
    ],
    targets: [
        .target(
            name: "CloudMockSDK",
            path: "Sources/CloudMockSDK"
        ),
    ]
)

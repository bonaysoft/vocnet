// swift-tools-version: 6.2
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "vocnet-sdk-swift",
    platforms: [
        .iOS(.v12),
        .macOS(.v10_15),
    ],
    products: [
        // Products define the executables and libraries a package produces, making them visible to other packages.
        .library(
            name: "vocnet-sdk-swift",
            targets: ["vocnet-sdk-swift"]
        ),
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-protobuf.git", from: "1.27.0"),
        .package(url: "https://github.com/connectrpc/connect-swift.git", from: "1.1.0")
    ],
    targets: [
        // Targets are the basic building blocks of a package, defining a module or a test suite.
        // Targets can depend on other targets in this package and products from dependencies.
        .target(
            name: "vocnet-sdk-swift",
            dependencies: [
                .product(name: "SwiftProtobuf", package: "swift-protobuf"),
                .product(name: "Connect", package: "connect-swift")
            ],
        ),
        .testTarget(
            name: "vocnet-sdk-swiftTests",
            dependencies: ["vocnet-sdk-swift"]
        ),
    ]
)

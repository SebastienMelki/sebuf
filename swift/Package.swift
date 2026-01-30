// swift-tools-version: 6.2
// The swift-tools-version declares the minimum version of Swift required to build this package.

@preconcurrency import PackageDescription

let package = Package(
	name: .name,
    products: [
        // Products define the executables and libraries a package produces, making them visible to other packages.
		.protocGenSwiftHTTP,
		.swiftSebuf
    ],
	dependencies: [
		.swiftProtobuf
	],
    targets: [
        // Targets are the basic building blocks of a package, defining a module or a test suite.
        // Targets can depend on other targets in this package and products from dependencies.
		.protocGenSwiftHTTP,
		.protocGenSwiftHTTPTest,
		
		.swiftSebuf,
		.swiftSebufTest
    ]
)

extension String {
	
	fileprivate static let name = "SwiftSebuf"
	
	// Modules
	fileprivate static let protocGenSwiftHTTP = "protoc-gen-swift-http"
	fileprivate static let swiftSebuf = "SwiftSebuf"
	
	// Packages
	fileprivate static let swiftProtobuf = "SwiftProtobuf"
	fileprivate static let swiftProtobufPluginLibrary = "SwiftProtobufPluginLibrary"
	fileprivate static let swiftProtobufPackage = "swift-protobuf"

	fileprivate var test: Self {
		"\(self)Tests"
	}
}

extension Product {

	fileprivate static let protocGenSwiftHTTP: Product = executable(
		name: .protocGenSwiftHTTP,
		targets: [
			.protocGenSwiftHTTP,
			.swiftSebuf
		]
	)
	fileprivate static let swiftSebuf: Product = library(
		name: .swiftSebuf,
		targets: [.swiftSebuf]
	)
}

extension Target {

	fileprivate static let protocGenSwiftHTTP: Target = executableTarget(
		name: .protocGenSwiftHTTP,
		dependencies: [
			.swiftProtobuf,
			.swiftProtobufPluginLibrary
		]
	)
	fileprivate static let protocGenSwiftHTTPTest: Target = testTarget(
		name: .protocGenSwiftHTTP.test,
		dependencies: [.protocGenSwiftHTTP],
		exclude: ["Golden/SimpleService/simple_user_service.proto"]
	)
	
	fileprivate static let swiftSebuf: Target = target(
		name: .swiftSebuf,
		dependencies: [.swiftProtobuf]
	)
	fileprivate static let swiftSebufTest: Target = testTarget(
		name: .swiftSebuf.test,
		dependencies: [.swiftSebuf]
	)
}

extension Target.Dependency {
	
	// Modules
	fileprivate static let protocGenSwiftHTTP: Target.Dependency = byName(name: .protocGenSwiftHTTP)
	fileprivate static let swiftSebuf: Target.Dependency = byName(name: .swiftSebuf)
	
	// Packages
	fileprivate static let swiftProtobuf: Target.Dependency = product(
		name: .swiftProtobuf,
		package: .swiftProtobufPackage
	)
	fileprivate static let swiftProtobufPluginLibrary: Target.Dependency = product(
		name: .swiftProtobufPluginLibrary,
		package: .swiftProtobufPackage
	)
}

extension Package.Dependency {
	
	fileprivate static let swiftProtobuf: Package.Dependency = package(
		url: "https://github.com/apple/swift-protobuf.git",
		exact: "1.33.3"
	)
}

// swift-tools-version: 6.2
// The swift-tools-version declares the minimum version of Swift required to build this package.

@preconcurrency import PackageDescription

let package = Package(
	name: .name,
	platforms: [.macOS(.v15)],
    products: [
        // Products define the executables and libraries a package produces, making them visible to other packages.
		.bufGenSwift
    ],
	dependencies: [
		.swiftArgumentParser,
		.swiftProtobuf
	],
    targets: [
        // Targets are the basic building blocks of a package, defining a module or a test suite.
        // Targets can depend on other targets in this package and products from dependencies.
		.bufGenSwift,
		.bufGenSwiftTest
    ]
)

extension String {
	
	fileprivate static let name: Self = "SwiftSebuf"
	
	// Modules
	fileprivate static let bufGenSwift: Self = "buf-gen-swift"
	
	// Packages
	fileprivate static let swiftArgumentParser: Self = "ArgumentParser"
	fileprivate static let swiftArgumentParserPackage: Self = "swift-argument-parser"
	fileprivate static let swiftProtobuf: Self = "SwiftProtobuf"
	fileprivate static let swiftProtobufPluginLibrary: Self = "SwiftProtobufPluginLibrary"
	fileprivate static let swiftProtobufPackage: Self = "swift-protobuf"

	fileprivate var test: Self {
		"\(self)Tests"
	}
}

extension Product {
	
	fileprivate static let bufGenSwift: Product = library(
		name: .bufGenSwift,
		targets: [.bufGenSwift]
	)
}

extension Target {
	
	fileprivate static let bufGenSwift: Target = target(
		name: .bufGenSwift,
		dependencies: [
			.swiftArgumentParser,
			.swiftProtobuf,
			.swiftProtobufPluginLibrary
		]
	)
	fileprivate static let bufGenSwiftTest: Target = testTarget(
		name: .bufGenSwift.test,
		dependencies: [.bufGenSwift]
	)
}

extension Target.Dependency {
	
	// Modules
	fileprivate static let bufGenSwift: Self = byName(name: .bufGenSwift)
	
	// Packages
	fileprivate static let swiftArgumentParser: Self = product(
		name: .swiftArgumentParser,
		package: .swiftArgumentParserPackage
	)
	fileprivate static let swiftProtobuf: Self = product(
		name: .swiftProtobuf,
		package: .swiftProtobufPackage
	)
	fileprivate static let swiftProtobufPluginLibrary: Self = product(
		name: .swiftProtobufPluginLibrary,
		package: .swiftProtobufPackage
	)
}

extension Package.Dependency {
	
	fileprivate static let swiftArgumentParser: Package.Dependency = package(
		url: "https://github.com/apple/swift-argument-parser.git",
		exact: "1.6.2"
	)
	fileprivate static let swiftProtobuf: Package.Dependency = package(
		url: "https://github.com/apple/swift-protobuf.git",
		exact: "1.33.3"
	)
}

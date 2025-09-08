// swift-tools-version: 6.1
// The swift-tools-version declares the minimum version of Swift required to build this package.

@preconcurrency import PackageDescription

let package = Package(
	name: .name,
	platforms: [.macOS(.v14)],
    products: [
        // Products define the executables and libraries a package produces, making them visible to other packages.
        .swiftSebuf
    ],
	dependencies: [
		.swiftArgumentParser,
		.swiftProtobuf
	],
    targets: [
        // Targets are the basic building blocks of a package, defining a module or a test suite.
        // Targets can depend on other targets in this package and products from dependencies.
        .swiftSebuf,
		.swiftSebufTest
    ]
)

private extension String {

	// MARK: Package Name
	static let name: Self = "SwiftSebuf"

	// MARK: Modules
	static let swiftSebuf: Self = "SwiftSebuf"

	// MARK: Packages
	static let swiftArgumentParser: Self = "ArgumentParser"
	static let swiftArgumentParserPackage: Self = "swift-argument-parser"
	static let swiftProtobuf: Self = "SwiftProtobuf"
	static let swiftProtobufPluginLibrary: Self = "SwiftProtobufPluginLibrary"
	static let swiftProtobufPackage: Self = "swift-protobuf"

	var test: Self {
		"\(self)Tests"
	}
}

private extension Product {

	static let swiftSebuf: Product = library(name: .swiftSebuf, targets: [.swiftSebuf])
}

private extension Target {

	static let swiftSebuf: Target = target(
		name: .swiftSebuf,
		dependencies: [
			.swiftArgumentParser,
			.swiftProtobuf,
			.swiftProtobufPluginLibrary
		]
	)
	static let swiftSebufTest: Target = testTarget(name: .swiftSebuf.test, dependencies: [.swiftSebuf])
}

private extension Target.Dependency {

	// MARK: Modules
	static let swiftSebuf: Self = byName(name: .swiftSebuf)

	// MARK: Packages
	static let swiftArgumentParser: Self = product(
		name: .swiftArgumentParser,
		package: .swiftArgumentParserPackage
	)
	static let swiftProtobuf: Self = product(name: .swiftProtobuf, package: .swiftProtobufPackage)
	static let swiftProtobufPluginLibrary: Self = product(
		name: .swiftProtobufPluginLibrary,
		package: .swiftProtobufPackage
	)
}

private extension Package.Dependency {

	static let swiftArgumentParser: Package.Dependency = package(
		url: "https://github.com/apple/swift-argument-parser.git",
		exact: "1.6.1"
	)
	static let swiftProtobuf: Package.Dependency = package(
		url: "https://github.com/apple/swift-protobuf.git",
		exact: "1.31.0"
	)
}

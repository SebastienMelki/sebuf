// swift-tools-version: 6.2

@preconcurrency import PackageDescription

let package = Package(
	name: .name,
	platforms: [.macOS(.v14)],
    products: [.swiftSebuf],
	dependencies: [
		.swiftArgumentParser,
		.swiftProtobuf
	],
    targets: [
		.swiftSebuf,
		.swiftSebufTest
    ]
)

extension String {
	
	fileprivate static let name: Self = "SwiftSebuf"
	
	// Modules
	fileprivate static let swiftSebuf: Self = "SwiftSebuf"
	
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
	
	fileprivate static let swiftSebuf: Product = library(name: .swiftSebuf, targets: [.swiftSebuf])
}

extension Target {

	fileprivate static let swiftSebuf: Target = target(
		name: .swiftSebuf,
		dependencies: [
			.swiftArgumentParser,
			.swiftProtobuf,
			.swiftProtobufPluginLibrary
		]
	)
	fileprivate static let swiftSebufTest: Target = testTarget(name: .swiftSebuf.test, dependencies: [.swiftSebuf])
}

extension Target.Dependency {

	// Modules
	fileprivate static let swiftSebuf: Self = byName(name: .swiftSebuf)

	// Packages
	fileprivate static let swiftArgumentParser: Self = product(
		name: .swiftArgumentParser,
		package: .swiftArgumentParserPackage
	)
	fileprivate static let swiftProtobuf: Self = product(name: .swiftProtobuf, package: .swiftProtobufPackage)
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

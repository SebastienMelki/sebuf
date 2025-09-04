// swift-tools-version: 6.1
// The swift-tools-version declares the minimum version of Swift required to build this package.

@preconcurrency import PackageDescription

let package = Package(
	name: .name,
    products: [
        // Products define the executables and libraries a package produces, making them visible to other packages.
        .swiftSebuf
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

	// MARK: Plugins
//	static let swiftLintPlugin: Self = "SwiftLintBuildToolPlugin"
//	static let swiftLintPackage: Self = "SwiftLintPlugins"

	var test: Self {
		"\(self)Tests"
	}
}

private extension Product {

	static let swiftSebuf: Product = .library(name: .swiftSebuf, targets: [.swiftSebuf])
}

private extension Target {

	static let swiftSebuf: Target = target(name: .swiftSebuf)
	static let swiftSebufTest: Target = testTarget(name: .swiftSebuf.test, dependencies: [.swiftSebuf])
}

private extension Target.Dependency {

	// MARK: Modules
	static let swiftSebuf: Self = byName(name: .swiftSebuf)

	// MARK: Packages
}

private extension Target.PluginUsage {

//	static let swiftLint: Self = plugin(name: .swiftLintPlugin, package: .swiftLintPackage)
}

private extension Package.Dependency {

//	static let swiftLint: Package.Dependency = package(url: "https://github.com/SimplyDanny/SwiftLintPlugins", exact: "0.59.1")
}

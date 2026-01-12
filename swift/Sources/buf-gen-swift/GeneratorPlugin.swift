//
//  GeneratorPlugin.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import SwiftProtobuf
import SwiftProtobufPluginLibrary

@main
internal struct GeneratorPlugin: CodeGenerator {
	
	internal func generate(
		files: [FileDescriptor],
		parameter: any CodeGeneratorParameter,
		protoCompilerContext: any ProtoCompilerContext,
		generatorOutputs: any GeneratorOutputs
	) throws {
		let options = try GeneratorOptions(parameter: parameter)
		for fileDescriptor in files {
			let fileGenerator = FileGenerator(descriptor: fileDescriptor, options: options)
			do {
				var printer = CodePrinter(addNewlines: true)
				try fileGenerator.generate(printer: &printer)
				try generatorOutputs.add(fileName: fileGenerator.name, contents: printer.content)
			} catch {
				throw GenerationError(error)
			}
		}
	}
	
	// MARK: Confirm with Codeowner on supported protobuf features
	internal let supportedFeatures: [Google_Protobuf_Compiler_CodeGeneratorResponse.Feature] = [
		.proto3Optional, .supportsEditions
	]
	// MARK: Confirm with Codeowner on the supported range
	internal let supportedEditionRange: ClosedRange<Google_Protobuf_Edition> = .proto3 ... .edition2024
	internal let version: String? = SwiftProtobuf.Version.versionString
	internal let projectURL: String? = "https://github.com/SebastienMelki/sebuf"
	internal let copyrightLine: String? = Version.copyright
	
	internal func printHelp() {
		// TODO: Figure out later, use swift-protobuf as reference
	}
}

/*
 
 TODO: Should be removed later, only for testing purposes
 
 Build Steps:
 
 swift build
 
 protoc \
	 --proto_path=Tests/buf-gen-swiftTests/Golden/SimpleService \
	 --swift_out=Tests/buf-gen-swiftTests/Golden/SimpleService \
	 --plugin=protoc-gen-swift=$(which protoc-gen-swift) \
	 Tests/buf-gen-swiftTests/Golden/SimpleService/simple_user_service.proto
 
 protoc \
	 --proto_path=Tests/buf-gen-swiftTests/Golden/SimpleService \
	 --swift-sebuf_out=Tests/buf-gen-swiftTests/Golden/SimpleService \
	 --plugin=protoc-gen-swift-sebuf=.build/debug/buf-gen-swift \
	 Tests/buf-gen-swiftTests/Golden/SimpleService/simple_user_service.proto
 
 */

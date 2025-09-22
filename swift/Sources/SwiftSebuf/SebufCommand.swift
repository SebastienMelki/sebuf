//
//  SebufCommand.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 08/09/2025.
//  Copyright © 2025 Sebuf. All rights reserved.
//

import ArgumentParser
import Foundation
import SwiftProtobuf
import SwiftProtobufPluginLibrary

protocol SebufCommand: ParsableCommand {

	associatedtype SebufCodeGenerator: Generator

	func makeResponse(_ request: CodeGeneratorRequest) -> CodeGeneratorResponse
}

extension SebufCommand {

	mutating func run() throws {
		guard let serializedBytes: Data = try FileHandle.standardInput.readToEnd() else { return }

		let request: CodeGeneratorRequest = try .init(serializedBytes: serializedBytes)

		let response: CodeGeneratorResponse = makeResponse(request)
		let responseData: Data = try response.serializedData()

		try FileHandle.standardOutput.write(contentsOf: responseData)
	}

	func makeResponse(_ request: CodeGeneratorRequest) -> CodeGeneratorResponse {
		let descriptorSet: DescriptorSet = .init(protos: request.protoFile)
		let generator: SebufCodeGenerator = .init(descriptorSet: descriptorSet)
		let response: CodeGeneratorResponse = generator.generate()
		return response
	}
}

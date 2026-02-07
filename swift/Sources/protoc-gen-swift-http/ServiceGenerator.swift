//
//  ServiceGenerator.swift
//  buf-gen-swift
//
//  Created by Khaled Chehabeddine on 10/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

internal final class ServiceGenerator {
	
	private let descriptor: ServiceDescriptor
	private let options: GeneratorOptions
	private let namer: SwiftProtobufNamer
	
	private let relativeName: String
	private let visibility: String
	
	internal init(descriptor: ServiceDescriptor, options: GeneratorOptions, namer: SwiftProtobufNamer) {
		self.descriptor = descriptor
		self.options = options
		self.namer = namer
		
		self.relativeName = namer.relativeName(service: descriptor)
		self.visibility = options.visibility.snippet
	}
	
	internal func generate(printer p: inout CodePrinter) {
		p.print()
		let commentsWithDeprecation = descriptor.protoSourceCommentsWithDeprecation(generatorOptions: options)
		p.print("\(commentsWithDeprecation)\(visibility)protocol \(relativeName): Sendable {")
		
		p.withIndentation { p in
			for method in descriptor.methods {
				method.generate(printer: &p, namer: namer)
			}
		}
		
		p.print("}")
	}
}

extension MethodDescriptor {
	
	fileprivate func generate(printer p: inout CodePrinter, namer: SwiftProtobufNamer) {
		let inputType = namer.relativeName(message: self.inputType)
		let outputType = namer.relativeName(message: self.outputType)
		let name = NamingUtils.toLowerCamelCase(self.name)
		p.print("func \(name)(_ request: \(inputType)) async throws -> \(outputType)")
	}
}

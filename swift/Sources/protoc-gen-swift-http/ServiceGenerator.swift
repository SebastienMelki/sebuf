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
	private let visibility: String
	
	internal init(descriptor: ServiceDescriptor, options: GeneratorOptions) {
		self.descriptor = descriptor
		self.options = options
		self.visibility = options.visibility.snippet
	}
	
	internal func generate(printer p: inout CodePrinter) {
		p.print()
		let commentsWithDeprecation = descriptor.protoSourceCommentsWithDeprecation(generatorOptions: options)
		p.print("\(commentsWithDeprecation)\(visibility)protocol \(descriptor.name): Sendable {")
		p.print()
		
		p.withIndentation { p in
			for method in descriptor.methods {
				method.generate(printer: &p)
			}
		}
		
		p.print("}")
	}
}

extension MethodDescriptor {
	
	fileprivate func generate(printer p: inout CodePrinter) {
		let inputType = self.inputType.name
		let outputType = self.outputType.name
		let name = self.name.camelCased()
		p.print("func \(name)(_ request: \(inputType)) async throws -> \(outputType)")
	}
}

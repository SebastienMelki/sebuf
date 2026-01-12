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
	private let visibility: String
	
	internal init(descriptor: ServiceDescriptor, options: GeneratorOptions) {
		self.descriptor = descriptor
		self.visibility = options.visibility.sourceSnippet
	}
	
	internal func generate(printer p: inout CodePrinter) {
		p.print()
		p.print("\(visibility)protocol \(descriptor.name): Sendable {")
		p.print()
		
		p.withIndentation { p in
			for method in descriptor.methods {
				generate(method: method, printer: &p)
			}
		}
		
		p.print("}")
	}
	
	private func generate(method: MethodDescriptor, printer p: inout CodePrinter) {
		let inputType = method.inputType.name
		let outputType = method.outputType.name
		let name = method.name.camelCased()
		p.print("func \(name)(_ request: \(inputType)) async throws -> \(outputType)")
	}
}

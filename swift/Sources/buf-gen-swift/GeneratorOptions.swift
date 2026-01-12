//
//  GeneratorOptions.swift
//  buf-gen-swift
//
//  Created by Khaled Chehabeddine on 10/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

internal final class GeneratorOptions {
	
	internal enum Visibility: String {
		
		case `internal`
		case `package`
		case `public`
		
		init?(flag: String) {
			self.init(rawValue: flag.lowercased())
		}
		
		var sourceSnippet: String {
			switch self {
			case .internal: ""
			case .package: "package "
			case .public: "public "
			}
		}
	}
	
	internal let visibility: Visibility
	
	internal init(parameter: any CodeGeneratorParameter) throws(GenerationError) {
		// TODO: Add options relevant to Sebuf, maybe use the same as swift-protobuf?
		// Backlog:
		// - File naming strategy
		// - Module mappings
		
		var visibility: Visibility = .internal
		
		for pair in parameter.parsedPairs {
			switch pair.key {
			case "Visibility":
				if let value = Visibility(flag: pair.value) {
					visibility = value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			default: throw .unknownParameter(name: pair.key)
			}
		}
		
		self.visibility = visibility
	}
}

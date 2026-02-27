//
//  GenerationError.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import Foundation

internal enum GenerationError: Error, CustomStringConvertible {
	
	case generator(GeneratorError)
	
	case invalidParameterValue(name: String, value: String)
	case unknownParameter(name: String)
	
	case message(String)
	case wrappedError(message: String, error: any Error)
	
	internal var description: String {
		switch self {
		case let .generator(error): error.description
		case let .invalidParameterValue(name, value): "Unknown value for generation parameter '\(name)': '\(value)'"
		case let .unknownParameter(name): "Unknown generation parameter '\(name)'"
		case let .message(message): message
		case let .wrappedError(message, error): "\(message): \(error)"
		}
	}
}

internal enum GeneratorError: Error, CustomStringConvertible {
	
	case invalidSwiftPrefix(filename: String, prefix: String)
	case message(String)
	
	internal var description: String {
		switch self {
		case let .invalidSwiftPrefix(filename, prefix):
			"\(filename) has an 'swift_prefix' that isn't a valid Swift identifier: \(prefix)."
		case let .message(message): message
		}
	}
}

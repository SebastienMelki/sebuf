//
//  GenerationError.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import Foundation

internal enum GenerationError: Error, CustomStringConvertible {
	
	case invalidParameterValue(name: String, value: String)
	case unknownParameter(name: String)
	case generator(GeneratorError)
	case unknown(any Error)
	
	internal init(_ error: any Error) {
		if let error = error as? GeneratorError {
			self = .generator(error)
		} else {
			self = .unknown(error)
		}
	}
	
	internal var description: String {
		switch self {
		case let .invalidParameterValue(name, value): "Unknown value for generation parameter '\(name)': '\(value)'"
		case let .unknownParameter(name): "Unknown generation parameter '\(name)'"
		case let .generator(error): error.description
		case let .unknown(error): error.localizedDescription
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

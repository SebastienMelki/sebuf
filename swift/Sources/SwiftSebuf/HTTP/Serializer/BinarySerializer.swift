//
//  BinarySerializer.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 29/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public struct BinarySerializer: Serializer {
	
	public let contentType = "application/x-protobuf"
	
	private let encodingOptions: BinaryEncodingOptions
	private let decodingOptions: BinaryDecodingOptions
	
	private let extensions: (any ExtensionMap)?
	private let partial: Bool
	
	public init(
		encodingOptions: BinaryEncodingOptions = .init(),
		decodingOptions: BinaryDecodingOptions = .init(),
		extensions: (any ExtensionMap)? = nil,
		partial: Bool = false
	) {
		self.encodingOptions = encodingOptions
		self.decodingOptions = decodingOptions
		self.extensions = extensions
		self.partial = partial
	}
	
	public func serialize<M: Message>(_ message: M) throws(SebufError) -> Data {
		do {
			return try message.serializedBytes(partial: partial, options: encodingOptions)
		} catch {
			throw .messageEncoding(error)
		}
	}
	
	public func deserialize<M: Message>(_ data: Data, as type: M.Type) throws(SebufError) -> M {
		do {
			return try M(serializedBytes: data, extensions: extensions, partial: partial, options: decodingOptions)
		} catch {
			throw .messageDecoding(error)
		}
	}
}

extension Serializer where Self == BinarySerializer {
	
	public static func binary(
		encodingOptions: BinaryEncodingOptions = .init(),
		decodingOptions: BinaryDecodingOptions = .init(),
		extensions: (any ExtensionMap)? = nil,
		partial: Bool = false
	) -> Self {
		BinarySerializer(
			encodingOptions: encodingOptions,
			decodingOptions: decodingOptions,
			extensions: extensions,
			partial: partial
		)
	}
}

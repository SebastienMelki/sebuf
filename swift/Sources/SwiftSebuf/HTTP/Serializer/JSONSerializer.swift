//
//  JSONSerializer.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 28/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public struct JSONSerializer: Serializer {
	
	public let contentType = "application/json"
	
	private let encodingOptions: JSONEncodingOptions
	private let decodingOptions: JSONDecodingOptions
	
	public init(encodingOptions: JSONEncodingOptions = .init(), decodingOptions: JSONDecodingOptions = .init()) {
		self.encodingOptions = encodingOptions
		self.decodingOptions = decodingOptions
	}
	
	public func serialize<M: Message>(_ message: M) throws(SebufError) -> Data {
		do {
			return try message.jsonUTF8Bytes(options: encodingOptions)
		} catch {
			throw .messageEncoding(error)
		}
	}
	
	public func deserialize<M: Message>(_ data: Data, as type: M.Type) throws(SebufError) -> M {
		do {
			return try M(jsonUTF8Bytes: data, options: decodingOptions)
		} catch {
			throw .messageDecoding(error)
		}
	}
}

extension Serializer where Self == JSONSerializer {
	
	public static func json(
		encodingOptions: JSONEncodingOptions = .init(),
		decodingOptions: JSONDecodingOptions = .init()
	) -> Self {
		JSONSerializer(encodingOptions: encodingOptions, decodingOptions: decodingOptions)
	}
}

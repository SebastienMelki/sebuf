//
//  Serializer.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 28/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public protocol Serializer: Sendable {
	
	func serialize<M: Message>(_ message: M) throws(SebufError) -> Data
	func deserialize<M: Message>(_ data: Data, as type: M.Type) throws(SebufError) -> M
}

//public protocol ContentSerializer: Sendable {
//
//	 /// The Content-Type header value for this serializer
//	 var contentType: String { get }
//
//	 /// Serialize a protobuf message to Data
//	 func serialize<M: Message>(_ message: M) throws -> Data
//
//	 /// Deserialize Data to a protobuf message
//	 func deserialize<M: Message>(_ data: Data, as type: M.Type) throws -> M
// }

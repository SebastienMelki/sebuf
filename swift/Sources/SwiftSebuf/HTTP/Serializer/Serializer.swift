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
	
	var contentType: String { get }
	
	func serialize<M: Message>(_ message: M) throws(SebufError) -> Data
	func deserialize<M: Message>(_ data: Data, as type: M.Type) throws(SebufError) -> M
}

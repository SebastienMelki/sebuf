//
//  Endpoint.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 23/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public protocol Endpoint: Configurable, Sendable {
	
	associatedtype Request: Message
	associatedtype Response: Message
	
	var configuration: ConfigurationValues { get }
	
	var path: String { get }
	var request: Request { get }
	
	func makeResponse() async throws(SebufError) -> Response
}

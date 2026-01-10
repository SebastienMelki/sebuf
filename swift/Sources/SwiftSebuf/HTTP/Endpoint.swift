//
//  Endpoint.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 23/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public protocol Endpoint: Configurable, Identifiable, Sendable {
	
	associatedtype Request: Message
	associatedtype Response: Message
	
	var id: String { get }
	
	var path: String { get }
	var request: Request { get }
	
	var response: Response { get async throws(SebufError) }
}

extension Endpoint {
	
	internal func makeTask(configuration: ConfigurationValues) -> NetworkTask<Self> {
		NetworkTask(configuration: configuration, endpoint: self)
	}
}

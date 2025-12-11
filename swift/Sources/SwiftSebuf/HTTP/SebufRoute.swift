//
//  Api.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public protocol SebufRoute: Sendable {
	
	associatedtype Request: Message
	associatedtype Response: Message
	
	var request: Request { get }
	var route: String { get }
}

extension SebufRoute {
	
//	public func data(from client: some SebufClient) async throws -> Response {
//		client.data(for: request)
//	}
}

//
//  SebufRoute.swift
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
	var path: String { get }
}

extension SebufRoute {
	
	internal func makeResponse(client: some SebufClient) async throws(SebufError) -> Response {
		let (data, _): (Data, URLResponse) = try await client.makeTask(route: self).value
		var options = JSONDecodingOptions()
		options.ignoreUnknownFields = true
		do {
			return try Response(jsonUTF8Bytes: data, options: options)
		} catch {
			throw .messageDecoding(error)
		}
	}
}

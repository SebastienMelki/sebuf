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
	
	public func resolve(in configuration: ConfigurationValues) async throws -> Response {
		let (data, _): (Data, URLResponse) = try await configuration.client.networkTask(for: self).value
		var options: JSONDecodingOptions = .init()
		options.ignoreUnknownFields = true
		return try Response(jsonUTF8Bytes: data, options: options)
	}
}

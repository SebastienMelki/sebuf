//
//  NetworkTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

public struct NetworkTask<Route: SebufRoute>: Sendable {
	
	private let configurations: ConfigurationValues
	private let client: any SebufClient
	private let route: Route
	
	init(configurations: ConfigurationValues, client: any SebufClient, route: Route) {
		self.configurations = configurations
		self.client = client
		self.route = route
	}
	
	public var value: (Data, URLResponse) {
		get async throws {
			let urlRequest: URLRequest = try route.makeURLRequest(configurations: configurations)
			let result: (Data, URLResponse) = try await client.session.data(for: urlRequest)
			return result
		}
	}
}

extension SebufRoute {
	
	fileprivate func makeURLRequest(configurations: ConfigurationValues) throws -> URLRequest {
		guard let baseURLString = configurations.baseURLString,
			  let url: URL = .init(string: baseURLString + route) else {
			throw SebufError.invalidURLRequest
		}
		var urlRequest: URLRequest = .init(url: url)
		urlRequest.httpMethod = "POST"
		
		var options: JSONEncodingOptions = .init()
		options.preserveProtoFieldNames = true
		urlRequest.httpBody = try request.jsonUTF8Data(options: options)
		
		return urlRequest
	}
}

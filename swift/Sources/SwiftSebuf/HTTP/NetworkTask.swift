//
//  NetworkTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

internal struct NetworkTask<Client: SebufClient, Route: SebufRoute>: Sendable {
	
	private let configurations: ConfigurationValues
	private let client: Client
	private let route: Route
	
	internal init(configurations: ConfigurationValues, client: Client, route: Route) {
		self.configurations = configurations
		self.client = client
		self.route = route
	}
	
	internal var value: (Data, URLResponse) {
		get async throws(SebufError) {
			let urlRequest: URLRequest = try route.makeURLRequest(configurations: configurations)
			do {
				let result: (Data, URLResponse) = try await client.session.data(for: urlRequest)
				return result
			} catch {
				throw SebufError(error)
			}
		}
	}
}

extension SebufRoute {
	
	fileprivate func makeURLRequest(configurations: ConfigurationValues) throws(SebufError) -> URLRequest {
		guard let url: URL = .init(string: configurations.baseURLString + route) else { throw .invalidURLRequest }
		
		var urlRequest: URLRequest = .init(url: url)
		urlRequest.httpMethod = "POST"
		
		var options: JSONEncodingOptions = .init()
		options.preserveProtoFieldNames = true
		do {
			urlRequest.httpBody = try request.jsonUTF8Data(options: options)
		} catch {
			throw .messageEncoding(error)
		}
		
		return urlRequest
	}
}

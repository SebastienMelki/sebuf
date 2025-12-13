//
//  NetworkTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public struct NetworkTask<Client: SebufClient, Route: SebufRoute>: Sendable {
	
	private let configurations: ConfigurationValues
	private let client: Client
	private let route: Route
	
	init(configurations: ConfigurationValues, client: Client, route: Route) {
		self.configurations = configurations
		self.client = client
		self.route = route
	}
	
	public func data() async throws -> (Data, URLResponse) {
		let urlRequest: URLRequest = try route.makeURLRequest(configurations: configurations)
		let result: (Data, URLResponse) = try await client.data(for: urlRequest)
		return result
	}
}

extension SebufRoute {
	
	fileprivate func makeURLRequest(configurations: ConfigurationValues) throws -> URLRequest {
		guard let baseURLString = configurations.baseURLString,
			  let url: URL = .init(string: baseURLString + route) else {
			throw SebufError.invalidURLRequest
		}
		var request = URLRequest(url: url)
		request.httpMethod = "POST"
		return request
	}
}

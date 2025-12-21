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
	
	private let client: Client
	private let route: Route
	
	internal init(client: Client, route: Route) {
		self.client = client
		self.route = route
	}
	
	internal var value: (Data, URLResponse) {
		get async throws(SebufError) {
			let urlRequest = try await makeURLRequest()
			do {
				let result: (Data, URLResponse) = try await client.configuration.session.data(for: urlRequest)
				return result
			} catch {
				throw SebufError(error)
			}
		}
	}
	
	private func makeURLRequest() async throws(SebufError) -> URLRequest {
		let configuration = await client.configuration
		guard let url = URL(string: configuration.baseURLString + route.path) else { throw .invalidURL }
		
		var urlRequest = URLRequest(url: url)
		urlRequest.httpMethod = "POST"
		
		var options = JSONEncodingOptions()
		options.preserveProtoFieldNames = true
		do {
			urlRequest.httpBody = try route.request.jsonUTF8Data(options: options)
		} catch {
			throw .messageEncoding(error)
		}
		return urlRequest
	}
}

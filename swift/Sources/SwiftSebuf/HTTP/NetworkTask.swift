//
//  NetworkTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

internal struct NetworkTask<E: Endpoint>: Sendable {
	
	private let configuration: ConfigurationValues
	
	private let endpoint: E
	private let session: URLSession
	
	internal init(endpoint: E, session: URLSession) {
		self.configuration = endpoint.configuration
		self.endpoint = endpoint
		self.session = session
	}
	
	internal var value: E.Response {
		get async throws(SebufError) {
			do {
				let urlRequest = try makeURLRequest()
				let (data, _): (Data, URLResponse) = try await session.data(for: urlRequest)
				return try configuration.serializer.deserialize(data, as: E.Response.self)
			} catch {
				throw SebufError(error)
			}
		}
	}
	
	private func makeURLRequest() throws(SebufError) -> URLRequest {
		guard let url = configuration.baseURL?.appending(path: endpoint.path) else { throw .invalidURL }
		
		var urlRequest = URLRequest(url: url)
		for modifier in configuration.requestModifiers {
			modifier.modify(&urlRequest)
		}
		urlRequest.httpBody = try configuration.serializer.serialize(endpoint.request)
		urlRequest.httpMethod = "POST"
		return urlRequest
	}
}

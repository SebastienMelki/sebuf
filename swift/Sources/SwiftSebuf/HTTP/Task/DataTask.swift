//
//  DataTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 03/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import Foundation

internal struct DataTask<E: Endpoint>: Sendable {
	
	private let configuration: ConfigurationValues
	private let endpoint: E
	
	internal init(configuration: ConfigurationValues, endpoint: E) {
		self.configuration = configuration
		self.endpoint = endpoint
	}
	
	internal var value: E.Response {
		get async throws(SebufError) {
			var statusCode: Int?
			do {
				let urlRequest = try makeURLRequest()
				configuration.logger?.logRequest(urlRequest, endpoint: endpoint.id)
				let (data, response) = try await configuration.client.session.data(for: urlRequest)
				if let httpResponse = response as? HTTPURLResponse {
					statusCode = httpResponse.statusCode
					configuration.logger?.logResponse(httpResponse, data: data, endpoint: endpoint.id)
					return try configuration.serializer.deserialize(data, as: E.Response.self)
				}
				throw URLError(.badServerResponse)
			} catch {
				if let statusCode, configuration.retryPolicy.retryableStatusCodes.contains(statusCode) {
					throw .retry
				}
				throw SebufError(error)
			}
		}
	}
	
	private func makeURLRequest() throws(SebufError) -> URLRequest {
		guard let url = configuration.baseURL?.appending(path: endpoint.path) else { throw .invalidURL }
		
		var urlRequest = URLRequest(
			url: url,
			cachePolicy: configuration.cachePolicy,
			timeoutInterval: configuration.timeoutInterval
		)
		for (key, value) in configuration.headers {
			urlRequest.setValue(key, forHTTPHeaderField: value)
		}
		urlRequest.setValue(configuration.serializer.contentType, forHTTPHeaderField: "Content-Type")
		urlRequest.httpBody = try configuration.serializer.serialize(endpoint.request)
		urlRequest.httpMethod = "POST"
		return urlRequest
	}
}

public struct _DataTask<Success: Endpoint>: Sendable {
	
	private let configuration: ConfigurationValues
	private let success: Success
	
	internal init(configuration: ConfigurationValues, success: Success) {
		self.configuration = configuration
		self.success = success
	}
	
//	public var value: Success.Response {
//		get async throws(SebufError) {
//
//		}
//	}
}

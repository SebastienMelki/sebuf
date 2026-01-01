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
			var latestError: (any Error)?
			for try await attempt in configuration.retryAsyncSequence {
				let endpoint = String(describing: E.self)
				do {
					let urlRequest = try makeURLRequest()
					configuration.logger?.logRequest(urlRequest, endpoint: endpoint)
					let (data, response) = try await session.data(for: urlRequest)
					if let httpResponse = response as? HTTPURLResponse {
						configuration.logger?.logResponse(httpResponse, data: data, endpoint: endpoint)
						if configuration.retryPolicy.retryableStatusCodes.contains(httpResponse.statusCode) {
							continue
						}
						return try configuration.serializer.deserialize(data, as: E.Response.self)
					}
					throw URLError(.badServerResponse)
				} catch {
					latestError = error
					configuration.logger?.logError(SebufError(error), endpoint: endpoint, attempt: attempt)
				}
			}
			throw SebufError(latestError ?? URLError(.unknown))
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
			urlRequest.setValue(value, forHTTPHeaderField: key)
		}
		// TODO: Debating whether or not to keep this for developer error safety...
//		for modifier in configuration.requestModifiers {
//			modifier.modify(&urlRequest)
//		}
		urlRequest.setValue(configuration.serializer.contentType, forHTTPHeaderField: "Content-Type")
		urlRequest.httpBody = try configuration.serializer.serialize(endpoint.request)
		urlRequest.httpMethod = "POST"
		return urlRequest
	}
}

extension ConfigurationValues {
	
	fileprivate var retryAsyncSequence: RetryAsyncSequence {
		RetryAsyncSequence(policy: retryPolicy)
	}
}

//
//  NetworkTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf

internal struct NetworkTask<Client: HTTPClient, E: Endpoint>: Sendable {
	
	private let client: Client
	private let endpoint: E
	
	internal init(client: Client, endpoint: E) {
		self.client = client
		self.endpoint = endpoint
	}
	
	internal var value: E.Response {
		get async throws(SebufError) {
			let urlRequest = try makeURLRequest()
			do {
				let (data, _): (Data, URLResponse) = try await client.session.data(for: urlRequest)
				var options = JSONDecodingOptions()
				options.ignoreUnknownFields = true
				return try E.Response(jsonUTF8Bytes: data, options: options)
			} catch {
				throw SebufError(error)
			}
		}
	}
	
	private func makeURLRequest() throws(SebufError) -> URLRequest {
		let configuration = endpoint.configuration
		guard let url = configuration.baseURL?.appending(path: endpoint.path) else { throw .invalidURL }
		
		var urlRequest = URLRequest(url: url)
		urlRequest.httpMethod = "POST"
		
		for modifier in configuration.requestModifiers {
			modifier.modify(&urlRequest)
		}
		
		var options = JSONEncodingOptions()
		options.preserveProtoFieldNames = true
		do {
			urlRequest.httpBody = try endpoint.request.jsonUTF8Data(options: options)
		} catch {
			throw .messageEncoding(error)
		}
		return urlRequest
	}
}

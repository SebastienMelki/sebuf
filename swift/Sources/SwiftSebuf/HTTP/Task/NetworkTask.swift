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
	
	internal init(configuration: ConfigurationValues, endpoint: E) {
		self.configuration = configuration
		self.endpoint = endpoint
	}
	
	internal var value: E.Response {
		get async throws(SebufError) {
			let dataTask = DataTask(configuration: configuration, endpoint: endpoint)
			for try await attempt in configuration.retryAsyncSequence {
				do {
					return try await dataTask.value
				} catch {
					configuration.logger?.logError(error, endpoint: endpoint.id, attempt: attempt)
					switch error {
					case .retry: continue
					default: throw error
					}
				}
			}
			throw SebufError(URLError(.unknown))
		}
	}
}

extension ConfigurationValues {
	
	fileprivate var retryAsyncSequence: RetryAsyncSequence {
		RetryAsyncSequence(policy: retryPolicy)
	}
}

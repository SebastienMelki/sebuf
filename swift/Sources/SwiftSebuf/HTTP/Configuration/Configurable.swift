//
//  Configurable.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 22/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol Configurable: Sendable {
	
	@discardableResult
	func configuration<V: Sendable>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self
}

extension Configurable {
	
	@discardableResult
	public func baseURL(_ url: URL?) -> Self {
		configuration(\.baseURL, url)
	}
	
	@discardableResult
	public func client(_ client: some HTTPClient) -> Self {
		configuration(\.client, client)
	}
	
	@discardableResult
	public func headers(_ headers: [String: String]) -> Self {
		configuration(\.headers, headers)
	}
	
	@discardableResult
	public func serializer(_ serializer: some Serializer) -> Self {
		configuration(\.serializer, serializer)
	}
}

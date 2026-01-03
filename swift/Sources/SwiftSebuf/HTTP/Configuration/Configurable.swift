//
//  Configurable.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 22/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol Configurable: Sendable {
	
	func configuration<V: Sendable>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self
}

extension Configurable {
	
	public func baseURL(_ url: URL?) -> Self {
		configuration(\.baseURL, url)
	}
	
	public func client(_ client: some HTTPClient) -> Self {
		configuration(\.client, client)
	}
	
	public func headers(_ headers: [String: String]) -> Self {
		configuration(\.headers, headers)
	}
	
	public func serializer(_ serializer: some Serializer) -> Self {
		configuration(\.serializer, serializer)
	}
}

//
//  Configurable.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 22/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol Configurable {
	
	var configuration: ConfigurationValues { get set }
	
	func configuration<V>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self
}

extension Configurable {
	
	public func configuration<V>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self {
		var modified = self
		modified.configuration[keyPath: keyPath] = value
		return modified
	}
	
	public func baseURL(_ url: URL?) -> Self {
		configuration(\.baseURL, url)
	}
	
	public func requestModifiers(_ modifiers: [any RequestModifier]) -> Self {
		configuration(\.requestModifiers, modifiers)
	}
	
	public func serializer(_ serializer: some Serializer) -> Self {
		configuration(\.serializer, serializer)
	}
}

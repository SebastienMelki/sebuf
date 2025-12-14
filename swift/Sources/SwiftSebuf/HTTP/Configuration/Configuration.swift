//
//  Configuration.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol ConfigurationKey: Sendable {
	
	associatedtype Value: Sendable
	
	static var defaultValue: Value { get }
}

public struct ConfigurationValues: Sendable {
	
	private var values: [String: any Sendable] = [:]

	public init() {
	}
	
	public subscript<K: ConfigurationKey>(key: K.Type) -> K.Value {
		get {
			let keyString: String = .init(reflecting: key)
			guard let value: K.Value = values[keyString] as? K.Value else { return K.defaultValue }

			return value
		}
		set {
			let keyString: String = .init(reflecting: key)
			values[keyString] = newValue
		}
	}
}

@propertyWrapper public struct Configurations: Sendable {
	
	private var values: ConfigurationValues
	
	public init() {
		self.values = .init()
	}
	
	public var wrappedValue: ConfigurationValues {
		values
	}
	
	mutating func update<V>(_ value: V, for keyPath: WritableKeyPath<ConfigurationValues, V>) {
		self.values[keyPath: keyPath] = value
	}
}

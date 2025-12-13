//
//  Configuration.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftUI

public protocol ConfigurationKey: Sendable {
	
	associatedtype Value: Sendable
	
	static var defaultValue: Value { get }
}

public struct ConfigurationValues: Sendable {
	
	private var values: [ObjectIdentifier: any Sendable] = [:]
	
	public init() {
	}
	
	public subscript<K: ConfigurationKey>(key: K.Type) -> K.Value {
		get {
			guard let value: K.Value = values[ObjectIdentifier(key)] as? K.Value else { return K.defaultValue }
			
			return value
		}
		set {
			values[ObjectIdentifier(key)] = newValue
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

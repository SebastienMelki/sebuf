//
//  ConfigurationValues.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 17/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public struct ConfigurationValues: Sendable {
	
	private var storage = Storage()

	public init() {
	}
	
	public subscript<K: ConfigurationKey>(key: K.Type) -> K.Value {
		get {
			if let value: K.Value = storage[ObjectIdentifier(key)] as? K.Value {
				return value
			}
			return K.defaultValue
		}
		set {
			if !isKnownUniquelyReferenced(&storage) {
				storage = Storage(values: storage.allValues)
			}
			storage[ObjectIdentifier(key)] = newValue
		}
	}
}

extension ConfigurationValues {
	
	private final class Storage: @unchecked Sendable {
		
		private var lock = os_unfair_lock_s()
		
		private var values: [ObjectIdentifier: any Sendable] = [:]
		
		fileprivate init(values: [ObjectIdentifier: any Sendable] = [:]) {
			self.values = values
		}
		
		fileprivate subscript(key: ObjectIdentifier) -> (any Sendable)? {
			get {
				withLock {
					values[key]
				}
			}
			set {
				withLock {
					values[key] = newValue
				}
			}
		}
		
		fileprivate var allValues: [ObjectIdentifier: any Sendable] {
			withLock {
				values
			}
		}
		
		private func withLock<R: Sendable>(_ body: () -> R) -> R {
			os_unfair_lock_lock(&lock)
			defer { os_unfair_lock_unlock(&lock) }
			return body()
		}
	}
}

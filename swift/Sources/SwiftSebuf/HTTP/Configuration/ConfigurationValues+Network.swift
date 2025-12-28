//
//  ConfigurationValues+Network.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 17/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

// TODO: Add remaining configuration values
extension ConfigurationValues {
	
	public var baseURL: URL? {
		get {
			self[BaseURLConfigurationKey.self]
		}
		set {
			self[BaseURLConfigurationKey.self] = newValue
		}
	}
	
	public var requestModifiers: [any RequestModifier] {
		get {
			self[RequestModifiersConfigurationKey.self]
		}
		set {
			self[RequestModifiersConfigurationKey.self] = newValue
		}
	}
	
	public var serializer: any Serializer {
		get {
			self[SerializerConfigurationKey.self]
		}
		set {
			self[SerializerConfigurationKey.self] = newValue
		}
	}
}

private struct BaseURLConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: URL? = nil
}

private struct RequestModifiersConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: [any RequestModifier] = []
}

private struct SerializerConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: any Serializer = .json()
}

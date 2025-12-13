//
//  ConfigurationValues+Customization.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 13/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

// TODO: Add the base configuration values
extension ConfigurationValues {
	
	public var baseURLString: String? {
		get {
			self[BaseURLStringConfigurationKey.self]
		}
		set {
			self[BaseURLStringConfigurationKey.self] = newValue
		}
	}
	
	// TODO: Finalize SebufClient protocol and default implementation
	public var client: any SebufClient {
		get {
			self[ClientConfigurationKey.self]
		}
		set {
			self[ClientConfigurationKey.self] = newValue
		}
	}
}

private struct BaseURLStringConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: String? = nil
}

private struct ClientConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: any SebufClient = DefaultSebufClient()
}

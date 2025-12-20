//
//  ConfigurationValues+Network.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 17/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

// TODO: Add the base configuration values
extension ConfigurationValues {
	
	public var baseURLString: String {
		get {
			self[BaseURLStringConfigurationKey.self]
		}
		set {
			self[BaseURLStringConfigurationKey.self] = newValue
		}
	}
}

private struct BaseURLStringConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: String = ""
}

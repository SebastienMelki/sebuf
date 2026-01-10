//
//  ConfigurationKey.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 17/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol ConfigurationKey: Sendable {
	
	associatedtype Value: Sendable
	
	static var defaultValue: Value { get }
}

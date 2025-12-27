//
//  HeaderModifier.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 21/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public struct HeaderModifier: RequestModifier {
	
	private let key: String
	private let value: String?
	
	public init(key: String, value: String?) {
		self.key = key
		self.value = value
	}
	
	public func modify(_ request: inout URLRequest) {
		request.setValue(value, forHTTPHeaderField: key)
	}
}

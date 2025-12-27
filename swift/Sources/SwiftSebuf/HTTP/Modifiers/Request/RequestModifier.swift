//
//  RequestModifier.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 21/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol RequestModifier: Sendable {
	
	func modify(_ request: inout URLRequest)
}

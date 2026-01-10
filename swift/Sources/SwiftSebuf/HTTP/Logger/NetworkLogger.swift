//
//  NetworkLogger.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 01/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol NetworkLogger: Sendable {
	
	func logRequest(_ request: URLRequest, endpoint: String)
	func logResponse(_ response: HTTPURLResponse, data: Data, endpoint: String)
	func logError(_ error: SebufError, endpoint: String, attempt: Int)
}

//
//  DefaultLogger.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 01/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import Foundation
import OSLog

public struct DefaultLogger: NetworkLogger {
	
	public enum Level: Sendable {
		case error
		case info
		case debug
	}
	
	private let logger = Logger(subsystem: "package.swift-sebuf", category: "Network")
	
	private let level: Level?
	
	public init(level: Level?) {
		self.level = level
	}
	
	public func logRequest(_ request: URLRequest, endpoint: String) {
		logger.info("")
	}
	
	public func logResponse(_ response: HTTPURLResponse, data: Data, endpoint: String) {
		logger.info("")
	}
	
	public func logError(_ error: SebufError, endpoint: String, attempt: Int) {
		logger.error("")
	}
}

extension NetworkLogger where Self == DefaultLogger {
	
	public static func `default`(level: DefaultLogger.Level? = nil) -> Self {
		DefaultLogger(level: level)
	}
}

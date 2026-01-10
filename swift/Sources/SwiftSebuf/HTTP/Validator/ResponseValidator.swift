//
//  ResponseValidator.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 01/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import Foundation

public struct ResponseValidator: Sendable {
	
	public let validStatusCodes: ClosedRange<Int>
	public let validateContentType: Bool
	
	public init(validStatusCodes: ClosedRange<Int>, validateContentType: Bool) {
		self.validStatusCodes = validStatusCodes
		self.validateContentType = validateContentType
	}
}

extension ResponseValidator {
	
	public static let `default` = ResponseValidator(validStatusCodes: 200...299, validateContentType: true)
	public static let permissive = ResponseValidator(validStatusCodes: 200...599, validateContentType: false)
	public static let strict = ResponseValidator(validStatusCodes: 200...200, validateContentType: true)
}

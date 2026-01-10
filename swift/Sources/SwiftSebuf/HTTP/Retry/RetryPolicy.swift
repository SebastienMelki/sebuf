//
//  RetryPolicy.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 29/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public struct RetryPolicy: Sendable {
	
	public enum BackoffStrategy: Sendable {
		case instant
		case constant(Duration)
		case exponential(duration: Duration, multiplier: Double)
		case custom(@Sendable (_ attempt: Int) -> Duration)
	}
	
	public let maxAttempts: Int
	public let retryableStatusCodes: Set<Int>
	public let backoffStrategy: BackoffStrategy
	
	public init(maxAttempts: Int, retryableStatusCodes: Set<Int>, backoffStrategy: BackoffStrategy) {
		self.maxAttempts = maxAttempts
		self.retryableStatusCodes = retryableStatusCodes
		self.backoffStrategy = backoffStrategy
	}
}

extension RetryPolicy {
	
	public static var none: RetryPolicy {
		RetryPolicy(maxAttempts: 1, retryableStatusCodes: [], backoffStrategy: .instant)
	}
}

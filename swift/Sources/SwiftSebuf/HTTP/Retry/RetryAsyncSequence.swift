//
//  RetryAsyncSequence.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 31/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

internal struct RetryAsyncSequence: AsyncSequence {
	
	internal struct Iterator: AsyncIteratorProtocol {
		
		private var attempt: Int = 0
		
		internal let policy: RetryPolicy
		
		fileprivate init(policy: RetryPolicy) {
			self.policy = policy
		}
		
		internal mutating func next() async throws(SebufError) -> Int? {
			guard attempt < policy.maxAttempts else { return nil }
			defer { attempt += 1 }
			
			if attempt > 0 {
				do {
					try await Task.sleep(for: duration)
				} catch {
					throw SebufError(error)
				}
			}
			return attempt + 1
		}
		
		private var duration: Duration {
			switch policy.backoffStrategy {
			case .instant: .zero
			case let .constant(duration): duration
			case let .exponential(duration, multiplier): exponentialDuration(duration: duration, multiplier: multiplier)
			case let .custom(durationFactory): durationFactory(attempt)
			}
		}
		
		private func exponentialDuration(duration: Duration, multiplier: Double) -> Duration {
			let scale = pow(multiplier, Double(attempt))
			guard scale.isFinite else { return .maximum }
			
			return Duration(
				secondsComponent: duration.components.seconds.scale(scale),
				attosecondsComponent: duration.components.attoseconds.scale(scale)
			)
		}
	}
	
	internal let policy: RetryPolicy
	
	internal func makeAsyncIterator() -> Iterator {
		Iterator(policy: policy)
	}
}

extension Duration {
	
	fileprivate static let maximum = Duration(secondsComponent: .max, attosecondsComponent: .max)
}

extension Int64 {
	
	fileprivate func scale(_ scale: Double) -> Int64 {
		let value = Double(self) * scale
		return value < Double(Int64.max) ? Int64(value) : .max
	}
}

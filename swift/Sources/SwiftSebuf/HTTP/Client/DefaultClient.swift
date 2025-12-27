//
//  DefaultClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 27/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

// TODO: Fix the locking logic, refer to hmlongco/Factory's implementation for a better idea?
public actor DefaultClient: HTTPClient {
	
	private nonisolated(unsafe) var lock = os_unfair_lock_s()
	
	public nonisolated(unsafe) var configuration: ConfigurationValues
	
	public let session: URLSession
	
	public init(configuration: ConfigurationValues = .init(), session: URLSession = .shared) {
		self.configuration = configuration
		self.session = session
	}
	
	public nonisolated func configuration<V>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self {
		withLock {
			configuration[keyPath: keyPath] = value
		}
		return self
	}
	
	private nonisolated func withLock<R: Sendable>(_ body: () -> R) -> R {
		os_unfair_lock_lock(&lock)
		defer {
			os_unfair_lock_unlock(&lock)
		}
		return body()
	}
}

extension HTTPClient where Self == DefaultClient {
	
	public static func `default`(configuration: ConfigurationValues = .init(), session: URLSession = .shared) -> Self {
		DefaultClient(configuration: configuration, session: session)
	}
}

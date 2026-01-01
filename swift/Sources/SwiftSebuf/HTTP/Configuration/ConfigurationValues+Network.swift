//
//  ConfigurationValues+Network.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 17/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

// TODO: Add remaining configuration values
extension ConfigurationValues {
	
	public internal(set) var baseURL: URL? {
		get {
			self[BaseURLConfigurationKey.self]
		}
		set {
			self[BaseURLConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var cachePolicy: URLRequest.CachePolicy {
		get {
			self[CachePolicyConfigurationKey.self]
		}
		set {
			self[CachePolicyConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var headers: [String: String] {
		get {
			self[HeadersConfigurationKey.self]
		}
		set {
			self[HeadersConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var logger: (any NetworkLogger)? {
		get {
			self[LoggerConfigurationKey.self]
		}
		set {
			self[LoggerConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var responseValidator: ResponseValidator {
		get {
			self[ResponseValidatorConfigurationKey.self]
		}
		set {
			self[ResponseValidatorConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var retryPolicy: RetryPolicy {
		get {
			self[RetryPolicyConfigurationKey.self]
		}
		set {
			self[RetryPolicyConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var serializer: any Serializer {
		get {
			self[SerializerConfigurationKey.self]
		}
		set {
			self[SerializerConfigurationKey.self] = newValue
		}
	}
	
	public internal(set) var timeoutInterval: TimeInterval {
		get {
			self[TimeoutIntervalConfigurationKey.self]
		}
		set {
			self[TimeoutIntervalConfigurationKey.self] = newValue
		}
	}
}

private struct BaseURLConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: URL? = nil
}

private struct CachePolicyConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: URLRequest.CachePolicy = .useProtocolCachePolicy
}

private struct HeadersConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: [String: String] = [:]
}

private struct LoggerConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: (any NetworkLogger)? = nil
}

private struct ResponseValidatorConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: ResponseValidator = .default
}

private struct RetryPolicyConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: RetryPolicy = .none
}

private struct SerializerConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: any Serializer = .json()
}

private struct TimeoutIntervalConfigurationKey: ConfigurationKey {
	
	fileprivate static let defaultValue: TimeInterval = 60.0
}

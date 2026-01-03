//
//  UserService.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 27/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf
@testable import SwiftSebuf

public struct UserService: Configurable {
	
	private var configuration: ConfigurationValues
	
	public init(configuration: ConfigurationValues = .init()) {
		self.configuration = configuration
	}
	
	public func configuration<V: Sendable>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self {
		var modified = self
		modified.configuration[keyPath: keyPath] = value
		return modified
	}
}

extension UserService {
	
	public struct CreateUserEndpoint: Endpoint {
		
		public typealias Request = CreateUserRequest
		public typealias Response = CreateUserResponse
		
		private var configuration: ConfigurationValues
		
		public let id: String = "UserService-CreateUserEndpoint"
		
		public let path: String = "/user/create"
		public let request: Request
		
		fileprivate init(configuration: ConfigurationValues, request: Request) {
			self.configuration = configuration
			self.request = request
		}
		
		public var response: Response {
			get async throws(SebufError) {
				try await makeTask(configuration: configuration).value
			}
		}
		
		public func configuration<V: Sendable>(_ keyPath: WritableKeyPath<ConfigurationValues, V>, _ value: V) -> Self {
			var modified = self
			modified.configuration[keyPath: keyPath] = value
			return modified
		}
	}
	
	public func createUser(_ request: CreateUserRequest) async throws(SebufError) -> CreateUserEndpoint {
		CreateUserEndpoint(configuration: configuration, request: request)
	}
}

// MARK: Generate this data from a proper proto file once tests are finalized

public struct CreateUserRequest: Message {
	
	public static let protoMessageName = "CreateUserRequest"
	
	public var unknownFields = UnknownStorage()
	
	public init() {
	}
	
	public mutating func decodeMessage<D: Decoder>(decoder: inout D) throws {
	}
	
	public func traverse<V: Visitor>(visitor: inout V) throws {
	}
	
	public func isEqualTo(message: any Message) -> Bool {
		true
	}
}

public struct CreateUserResponse: Message {
	
	public static let protoMessageName = "CreateUserResponse"
	
	public var unknownFields = UnknownStorage()
	
	let myCustomParameter = 3
	
	public init() {
	}
	
	public mutating func decodeMessage<D: Decoder>(decoder: inout D) throws {
	}
	
	public func traverse<V: Visitor>(visitor: inout V) throws {
	}
	
	public func isEqualTo(message: any Message) -> Bool {
		true
	}
}

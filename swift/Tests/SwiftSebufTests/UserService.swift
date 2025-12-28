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

public struct UserService<Client: HTTPClient>: Service {
	
	public var configuration: ConfigurationValues
	
	public let client: Client
	
	public init(client: Client) {
		self.configuration = client.configuration
		self.client = client
	}
}

extension UserService {
	
	private struct CreateUserEndpoint: Endpoint {
		
		typealias Request = CreateUserRequest
		typealias Response = CreateUserResponse
		
		var configuration: ConfigurationValues
		
		let path: String = "/user/create"
		let request: Request
		
		private let client: Client
		
		fileprivate init(configuration: ConfigurationValues, request: Request, client: Client) {
			self.configuration = configuration
			self.request = request
			self.client = client
		}
		
		var response: Response {
			get async throws(SebufError) {
				try await client.makeTask(endpoint: self).value
			}
		}
	}
	
	public func createUser(_ request: CreateUserRequest) async throws(SebufError) -> some Endpoint {
		CreateUserEndpoint(configuration: configuration, request: request, client: client)
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

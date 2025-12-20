//
//  SimpleService.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 14/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf
@testable import SwiftSebuf

public struct SimpleService<Client: SebufClient>: SebufService {
	
	private let client: Client
	
	private let headers: [String: String] = ["X-API-Key": "123e4567-e89b-12d3-a456-426614174000"]
	
	public init(client: Client) {
		self.client = client
	}
	
	private struct GetSimple: SebufRoute {
		
		typealias Request = GetSimpleRequest
		typealias Response = GetSimpleResponse
		
		let request: Request
		let route: String = "example/v1/simple/get"
		
		init(_ request: Request) {
			self.request = request
		}
	}
	
	public func getSimple(_ request: GetSimpleRequest) async throws(SebufError) -> GetSimpleResponse {
		try await GetSimple(request).response(from: client)
	}
}

// MARK: Generate this data from a proper proto file once tests are finalized

public struct GetSimpleRequest: Message {
	
	public static let protoMessageName: String = "GetSimpleRequest"
	
	public var unknownFields: UnknownStorage = .init()
	
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

public struct GetSimpleResponse: Message {
	
	public static let protoMessageName: String = "GetSimpleResponse"
	
	public var unknownFields: UnknownStorage = .init()
	
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

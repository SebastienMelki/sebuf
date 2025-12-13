//
//  SebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol SebufClient: Actor {
	
	var session: URLSession { get }
	
	func networkTask<Route: SebufRoute>(for route: Route) async throws(SebufError) -> NetworkTask<Self, Route>
}

extension SebufClient {
	
	func data(for request: URLRequest) async throws -> (Data, URLResponse) {
		try await session.data(for: request)
	}
}

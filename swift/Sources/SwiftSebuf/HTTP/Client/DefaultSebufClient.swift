//
//  DefaultSebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

actor DefaultSebufClient: SebufClient {
	
	let session: URLSession
	
	init(session: URLSession = .shared) {
		self.session = session
	}
	
	func data(for request: URLRequest) async throws(SebufError) -> (Data, URLResponse) {
		do {
			return try await session.data(for: request)
		} catch {
			throw SebufError(error)
		}
	}
	
	func networkTask<Route: SebufRoute>(
		for route: Route
	) async throws(SebufError) -> NetworkTask<DefaultSebufClient, Route> {
		NetworkTask(configurations: ConfigurationValues(), client: self, route: route)
	}
}

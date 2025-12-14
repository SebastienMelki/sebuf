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
	
	private let configurations: ConfigurationValues
	
	init(session: URLSession = .shared) {
		self.session = session
		self.configurations = .init()
	}
	
	func networkTask<Route: SebufRoute>(for route: Route) -> NetworkTask<Route> {
		NetworkTask(configurations: configurations, client: self, route: route)
	}
}

//
//  DataTask.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public struct DataTask<Client: SebufClient, Route: SebufRoute> {
	
	private let client: Client
	private let route: Route
	
	init(client: Client, route: Route) {
		self.client = client
		self.route = route
	}
	
//	public func data() async throws(SebufError) -> (Data, URLResponse) {
//		guard let url: URL = URL(string: <#T##String#>)
		
//		let urlRequest = route.request
		
//		let result = client.data(for: urlRequest)
//	}
}

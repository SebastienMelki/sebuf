//
//  SebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol SebufClient: Actor {
	
	associatedtype ClientError: Error
	
	var session: URLSession { get }
	
	func data(for request: URLRequest) async throws(ClientError) -> (Data, URLResponse)
	func dataTask<Route: SebufRoute>(for route: Route) async throws(ClientError) -> DataTask<Self, Route>
}

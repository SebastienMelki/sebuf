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
	
	func networkTask<Route: SebufRoute>(for route: Route) -> NetworkTask<Route>
}

//
//  SebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol SebufClient: Actor {
	
	var configurations: ConfigurationValues { get }
	var session: URLSession { get }
	
	nonisolated func service<S: SebufService>(_ type: S.Type) -> S where S.Client == Self
}

extension SebufClient {
	
	internal func networkTask<Route: SebufRoute>(for route: Route) -> NetworkTask<Self, Route> {
		NetworkTask(configurations: configurations, client: self, route: route)
	}
}

//
//  SebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol SebufClient: Actor {
	
	var configuration: ConfigurationValues { get }
	
	nonisolated func makeService<S: SebufService>(_ type: S.Type) -> S where S.Client == Self
}

extension SebufClient {
	
	internal func makeTask<Route: SebufRoute>(route: Route) -> NetworkTask<Self, Route> {
		NetworkTask(client: self, route: route)
	}
}

//
//  HTTPClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 23/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol HTTPClient: Actor, Configurable {
	
	nonisolated var configuration: ConfigurationValues { get }
	
	var session: URLSession { get }
}

extension HTTPClient {
	
	public func makeService<S: Service>(_ type: S.Type) -> S where S.Client == Self {
		S(client: self)
	}
	
	internal func makeTask<E: Endpoint>(endpoint: E) -> NetworkTask<Self, E> {
		NetworkTask(client: self, endpoint: endpoint)
	}
}

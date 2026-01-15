//
//  HTTPClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 23/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol HTTPClient: Actor {
	
	var session: URLSession { get }
}

//extension HTTPClient {
//	
//	public func makeTask<E: Endpoint>(endpoint: E) -> _DataTask<E> {
//		_DataTask(configuration: <#T##ConfigurationValues#>, success: endpoint)
//	}
//}

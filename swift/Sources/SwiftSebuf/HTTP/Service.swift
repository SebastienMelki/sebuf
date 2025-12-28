//
//  Service.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 23/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public protocol Service: Configurable, Sendable {
	
	associatedtype Client: HTTPClient
	
	var client: Client { get }
	
	init(client: Client)
}

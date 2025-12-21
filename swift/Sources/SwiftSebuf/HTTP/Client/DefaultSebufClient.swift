//
//  DefaultSebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public actor DefaultSebufClient: SebufClient {
	
	public let configuration: ConfigurationValues
	
	public init(configuration: ConfigurationValues = ConfigurationValues()) {
		self.configuration = configuration
	}
	
	public nonisolated func makeService<S: SebufService>(_ type: S.Type) -> S where S.Client == DefaultSebufClient {
		S(client: self)
	}
}

extension SebufClient where Self == DefaultSebufClient {
	
	public static func `default`(configuration: ConfigurationValues = ConfigurationValues()) -> Self {
		DefaultSebufClient(configuration: configuration)
	}
}

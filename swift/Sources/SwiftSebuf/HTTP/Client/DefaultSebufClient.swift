//
//  DefaultSebufClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public actor DefaultSebufClient: SebufClient {
	
	public let configurations: ConfigurationValues
	public let session: URLSession
	
	public init(configurations: ConfigurationValues = ConfigurationValues(), session: URLSession = .shared) {
		self.session = session
		self.configurations = configurations
	}
	
	public nonisolated func service<S: SebufService>(_ type: S.Type) -> S where S.Client == DefaultSebufClient {
		S(client: self)
	}
}

extension SebufClient where Self == DefaultSebufClient {
	
	public static func `default`(
		configurations: ConfigurationValues = ConfigurationValues(),
		session: URLSession = .shared
	) -> Self {
		DefaultSebufClient(configurations: configurations, session: session)
	}
}

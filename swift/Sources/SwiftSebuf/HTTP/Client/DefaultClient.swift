//
//  DefaultClient.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 27/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public actor DefaultClient: HTTPClient {
	
	public let session: URLSession
	
	public init(session: URLSession = .shared) {
		self.session = session
	}
}

extension HTTPClient where Self == DefaultClient {
	
	public static func `default`(session: URLSession = .shared) -> Self {
		DefaultClient(session: session)
	}
}

//
//  String+Utils.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

extension String {
	
	internal func camelCased() -> Self {
		if let char = self.first {
			return char.lowercased() + self.dropFirst()
		}
		return self
	}
}

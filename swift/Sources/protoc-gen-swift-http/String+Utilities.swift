//
//  String+Utilities.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

extension String {
	
	internal var pathComponents: (directory: String, base: String, extension: String) {
		// Separate directory from filename
		let lastSlash = self.lastIndex(of: "/")
		let directory = lastSlash.map { String(self[...$0]) } ?? ""
		let filename = lastSlash.map { String(self[self.index(after: $0)...]) } ?? self
		
		// Separate base from extension
		let lastDot = filename.lastIndex(of: ".")
		let base = lastDot.map { String(filename[..<$0]) } ?? filename
		let ext = lastDot.map { String(filename[$0...]) } ?? ""
		
		return (directory, base, ext)
	}
	
	internal func camelCased() -> Self {
		if let char = self.first {
			return char.lowercased() + self.dropFirst()
		}
		return self
	}
}

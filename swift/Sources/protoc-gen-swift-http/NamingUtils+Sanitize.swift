//
//  NamingUtils+Sanitize.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 18/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

extension NamingUtils {
	
	internal static func sanitize(serviceName s: String, forbiddenTypeNames: Set<String>) -> String {
		sanitizeTypeName(s, disambiguator: "Service", forbiddenTypeNames: forbiddenTypeNames)
	}
}

/// Sanitizes a type name to avoid Swift reserved words and naming conflicts.
///
/// - Parameters:
/// 	- s: The type name to sanitize
///		- disambiguator: Suffix to append when conflicts occur (e.g., "Service")
///		- forbiddenTypeNames: Additional names to treat as reserved (e.g., module names)
///	- Returns: The sanitized type name.
///	- Note: This code depends on the protoc validation of \_identifier\_, defined as [a-zA-z][a-zA-Z0-9_]. There is no
///			need for complex validation checks to handle characters outside these ranges.
private func sanitizeTypeName(_ s: String, disambiguator: String, forbiddenTypeNames: Set<String>) -> String {
	if reservedTypeNames.contains(s) || s.isOnlyUnderscore {
		return s + disambiguator
	} else if s.hasSuffix(disambiguator) {
		// There is a case where if `Foo` and `FooService` both exist, `Foo` can be expanded into `FooService`, therefore
		// `FooService` needs to become `FooServiceservice`. This is solved by recursively stripping the disambiguator,
		// sanitizing the root, then re-adding the disambiguator.
		let suffix = s.index(s.endIndex, offsetBy: -disambiguator.count)
		let truncated = String(s[..<suffix])
		return sanitizeTypeName(truncated, disambiguator: disambiguator, forbiddenTypeNames: forbiddenTypeNames)
		+ disambiguator
	} else if forbiddenTypeNames.contains(s) {
		// This case must run after the hasSuffix check. The forbidden type names set isn't fixed, and may contain values
		// like "FooService." Sanitization on these cases should prioritize suffix checks to avoid just appending the
		// disambiguator.
		// This is used for cases like module imports that are configurable (ex: renaming SwiftProtobuf).
		return s + disambiguator
	} else {
		return s
	}
}

private let reservedTypeNames: Set<String> = {
	var names: Set<String> = []

	// Framework namespaces - shadowing these blocks access to the frameworks.
	names.insert("Swift")
	names.insert("SwiftProtobuf")
	names.insert("SwiftSebuf")

	// Standard Swift property names.
	names.insert("debugDescription")
	names.insert("description")
	names.insert("dynamicType")
	names.insert("hashValue")

	// Keywords that interfere with type expressions.
	names.insert("Protocol")
	names.insert("Type")

	// Common protocols that could cause confusion.
	names.insert("Equatable")
	names.insert("Hashable")
	names.insert("Sendable")

	names.formUnion(swiftCommonTypes)
	names.formUnion(swiftKeywordsUsedInDeclarations)
	names.formUnion(swiftKeywordsUsedInExpressionsAndTypes)
	names.formUnion(swiftKeywordsUsedInStatements)
	names.formUnion(swiftSpecialVariables)
	
	return names
}()

extension String {
	
	fileprivate var isOnlyUnderscore: Bool {
		!self.isEmpty && self.allSatisfy { $0 == "_" }
	}
}

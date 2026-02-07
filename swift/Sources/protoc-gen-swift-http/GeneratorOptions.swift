//
//  GeneratorOptions.swift
//  buf-gen-swift
//
//  Created by Khaled Chehabeddine on 10/01/2026.
//  Copyright © 2026 Sebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

internal final class GeneratorOptions {
	
	internal enum FileNaming {
		
		case fullPath
		case pathToUnderscores
		case dropPath
		
		fileprivate init?(flag: String) {
			switch flag.lowercased() {
			case "fullpath", "full_path": self = .fullPath
			case "pathtounderscores", "path_to_underscores": self = .pathToUnderscores
			case "droppath", "drop_path": self = .dropPath
			default: return nil
			}
		}
	}
	
	internal enum ImportDirective {
		
		case accessLevel(Visibility)
		case plain
		case implementationOnly

		var isAccessLevel: Bool {
			switch self {
			case .accessLevel: true
			default: false
			}
		}

		var snippet: String {
			switch self {
			case let .accessLevel(visibility): "\(visibility.rawValue) import"
			case .plain: "import"
			case .implementationOnly: "@_implementationOnly import"
			}
		}
	}
	
	internal enum Visibility: String {
		
		case `internal`
		case `package`
		case `public`
		
		fileprivate init?(flag: String) {
			self.init(rawValue: flag.lowercased())
		}
		
		var snippet: String {
			switch self {
			case .internal: ""
			case .package: "package "
			case .public: "public "
			}
		}
	}
	
	internal let fileNaming: FileNaming
	internal let importDirective: ImportDirective
	internal let protoToModuleMappings: ProtoFileToModuleMappings
	internal let visibility: Visibility
	
	internal let experimentalStripNonfunctionalCodegen: Bool
	
	internal init(parameter: any CodeGeneratorParameter) throws(GenerationError) {
		var fileNaming: FileNaming = .fullPath
		var implementationOnlyImports = false
		var useAccessLevelOnImports = false
		var moduleMapPath: String?
		var swiftProtobufModuleName: String? = nil
		var visibility: Visibility = .internal
		
		var experimentalStripNonfunctionalCodegen = false
		
		for pair in parameter.parsedPairs {
			switch pair.key {
			case "FileNaming":
				if let naming = FileNaming(flag: pair.value) {
					fileNaming = naming
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			case "ImplementationOnlyImports":
				if let value = Bool(pair.value) {
					implementationOnlyImports = value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			case "UseAccessLevelOnImports":
				if let value = Bool(pair.value) {
					useAccessLevelOnImports = value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			case "ProtoPathModuleMappings":
				if !pair.value.isEmpty {
					moduleMapPath = pair.value
				}
			case "SwiftProtobufModuleName":
				if isValidSwiftIdentifier(pair.value) {
					swiftProtobufModuleName = pair.value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			case "Visibility":
				if let value = Visibility(flag: pair.value) {
					visibility = value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			case "experimental_strip_nonfunctional_codegen":
				if pair.value.isEmpty {
					experimentalStripNonfunctionalCodegen = true
				} else if let value = Bool(pair.value) {
					experimentalStripNonfunctionalCodegen = value
				} else {
					throw .invalidParameterValue(name: pair.key, value: pair.value)
				}
			default: throw .unknownParameter(name: pair.key)
			}
		}
		
		self.fileNaming = fileNaming
		
		switch (implementationOnlyImports, useAccessLevelOnImports) {
		case (false, false): self.importDirective = .plain
		case (false, true): self.importDirective = .accessLevel(visibility)
		case (true, false): self.importDirective = .implementationOnly
		case (true, true):
			throw .message(
				"""
				When using access levels on imports the @_implementationOnly option is unnecessary.
				Disable @_implementationOnly imports.
				"""
			)
		}
		
		if let moduleMapPath {
			do {
				self.protoToModuleMappings = try ProtoFileToModuleMappings(
					path: moduleMapPath,
					swiftProtobufModuleName: swiftProtobufModuleName
				)
			} catch {
				throw .wrappedError(message: "Parameter 'ProtoPathModuleMappings=\(moduleMapPath)'", error: error)
			}
		} else {
			self.protoToModuleMappings = ProtoFileToModuleMappings(swiftProtobufModuleName: swiftProtobufModuleName)
		}
		
		self.visibility = visibility
		
		self.experimentalStripNonfunctionalCodegen = experimentalStripNonfunctionalCodegen
		
		// Cross-check options for invalid combinations
		
		if implementationOnlyImports && visibility != .internal {
			throw .message(
				"""
				Cannot use @_implementationOnly imports when the proto visibility is public or package.
				Either change the visibility to internal, or disable @_implementationOnly imports.
				"""
			)
		}
	}
}

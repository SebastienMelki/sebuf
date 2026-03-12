//
//  SwiftProtobufNamer+Service.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 18/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

extension SwiftProtobufNamer {
	
	internal func relativeName(service: ServiceDescriptor) -> String {
		let prefix = typePrefix(forFile: service.file)
		return NamingUtils.sanitize(serviceName: prefix + service.name, forbiddenTypeNames: [self.swiftProtobufModuleName])
	}
	
	internal func fullName(service: ServiceDescriptor) -> String {
		let relativeName = self.relativeName(service: service)
		return modulePrefix(file: service.file) + relativeName
	}
	
	private func modulePrefix(file: FileDescriptor) -> String {
		guard let prefix = mappings.moduleName(forFile: file), prefix != targetModule else {
			return ""
		}
		return "\(prefix)."
	}
}

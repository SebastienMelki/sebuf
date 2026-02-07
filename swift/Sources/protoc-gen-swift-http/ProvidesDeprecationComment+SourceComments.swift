//
//  ProvidesDeprecationComment+SourceComments.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 15/01/2026.
//  Copyright © 2026 SwiftSebuf. All rights reserved.
//

import SwiftProtobufPluginLibrary

extension ProvidesDeprecationComment where Self: ProvidesSourceCodeLocation {
	
	internal func protoSourceCommentsWithDeprecation(
		generatorOptions: GeneratorOptions,
		commentPrefix: String = "///",
		leadingDetachedPrefix: String? = nil
	) -> String {
		if generatorOptions.experimentalStripNonfunctionalCodegen {
			return deprecationComment(commentPrefix: commentPrefix)
		}
		return protoSourceCommentsWithDeprecation(
			commentPrefix: commentPrefix,
			leadingDetachedPrefix: leadingDetachedPrefix
		)
	}
}

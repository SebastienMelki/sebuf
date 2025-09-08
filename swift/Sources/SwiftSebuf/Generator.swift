//
//  Generator.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 08/09/2025.
//  Copyright © 2025 Sebuf. All rights reserved.
//

import Foundation
import SwiftProtobuf
import SwiftProtobufPluginLibrary

protocol Generator {

	init(descriptorSet: DescriptorSet)

	func generate() -> CodeGeneratorResponse
}

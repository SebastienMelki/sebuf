//
//  OneofHelperCommand.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 04/09/2025.
//  Copyright © 2025 Sebuf. All rights reserved.
//

import ArgumentParser
import Foundation
import SwiftProtobuf
import SwiftProtobufPluginLibrary

@main
struct OneofHelperCommand: SebufCommand {

	typealias SebufCodeGenerator = OneofHelperGenerator

	static let configuration: CommandConfiguration = .init(
		commandName: "protoc-gen-swift-oneof-helper",
		abstract: "Generate Swift oneof helper extensions for protobuf messages"
	)
}

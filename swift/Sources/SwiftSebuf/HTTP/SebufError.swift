//
//  SebufError.swift
//  SwiftSebuf
//
//  Created by Khaled Chehabeddine on 11/12/2025.
//  Copyright © 2025 SwiftSebuf. All rights reserved.
//

import Foundation

public enum SebufError: Error {
	
	case cancelled
	case invalidURLRequest
	case urlError(URLError)
	case messageEncoding(any Error)
	case messageDecoding(any Error)
	case undefined(any Error)
	
	init(_ error: any Error) {
		if error is CancellationError {
			self = .cancelled
		} else if let error = error as? Self {
			self = error
		} else if let error = error as? URLError {
			self = .urlError(error)
		} else {
			self = .undefined(error)
		}
	}
}

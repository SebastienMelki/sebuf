// cmd/protoc-gen-swift-oneof-helper/main.go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Get the directory of this executable
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// Look for the Swift binary
	swiftBinary := filepath.Join(filepath.Dir(execPath), "protoc-gen-swift-oneof-helper-swift")

	// If Swift binary doesn't exist, try building it
	if _, err := os.Stat(swiftBinary); os.IsNotExist(err) {
		buildSwiftBinary(swiftBinary)
	}

	// Execute the Swift binary
	cmd := exec.Command(swiftBinary)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func buildSwiftBinary(outputPath string) {
	cmd := exec.Command("swift", "build", "-c", "release", "--product", "protoc-gen-swift-oneof-helper")
	if err := cmd.Run(); err != nil {
		panic("Failed to build Swift binary: " + err.Error())
	}

	// Copy the built binary to the expected location
	builtBinary := ".build/release/protoc-gen-swift-oneof-helper"
	cmd = exec.Command("cp", builtBinary, outputPath)
	cmd.Run()
}

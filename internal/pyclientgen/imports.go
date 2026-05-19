package pyclientgen

// writeImports emits the import block. Imports are stable regardless of file content
// to keep golden tests simple — unused imports in Python are not an error and Python
// linters treating them as such belong to user code, not generated code.
func writeImports(p printer, _ *collectedTypes) {
	p("from __future__ import annotations")
	p("")
	p("import base64")
	p("import binascii")
	p("import json")
	p("import urllib.error")
	p("import urllib.parse")
	p("import urllib.request")
	p("from dataclasses import dataclass, field")
	p("from datetime import datetime, timezone")
	p("from enum import IntEnum")
	p("from typing import Any, AsyncIterator, Iterator, Mapping, Optional, Protocol, Sequence, Union")
	p("")
}

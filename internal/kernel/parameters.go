/*
SPDX-FileCopyrightText: Copyright 2024 SAP SE or an SAP affiliate company and cobaltcore-dev contributors
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package kernel provides functions to read kernel parameters from the system.
package kernel

import (
	"os"
	"strings"
)

// DefaultCmdlinePath is the default path to the kernel command line.
const DefaultCmdlinePath = "/proc/cmdline"

// Parameters holds kernel boot parameters.
type Parameters struct {
	// CommandLine contains the raw kernel boot parameters from /proc/cmdline.
	CommandLine string
}

// Interface provides an interface for reading kernel parameters.
type Interface interface {
	// ReadParameters reads and returns kernel parameters from the system.
	ReadParameters() (*Parameters, error)
}

// SystemReader reads kernel parameters from the actual system files.
type SystemReader struct {
	cmdlinePath string
}

// NewSystemReader creates a new SystemReader with the default cmdline path.
func NewSystemReader() *SystemReader {
	return &SystemReader{
		cmdlinePath: DefaultCmdlinePath,
	}
}

// NewSystemReaderWithPath creates a new SystemReader with a custom cmdline path.
// This is useful for testing.
func NewSystemReaderWithPath(cmdlinePath string) *SystemReader {
	return &SystemReader{
		cmdlinePath: cmdlinePath,
	}
}

// ReadParameters reads kernel parameters from /proc/cmdline and returns them
// as a Parameters struct with the raw command line content.
func (r *SystemReader) ReadParameters() (*Parameters, error) {
	data, err := os.ReadFile(r.cmdlinePath)
	if err != nil {
		return nil, err
	}

	cmdline := strings.TrimSpace(string(data))

	return &Parameters{CommandLine: cmdline}, nil
}

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

package kernel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadParameters(t *testing.T) {
	tests := []struct {
		name           string
		cmdlineContent string
		expectedParams string
		expectError    bool
	}{
		{
			name:           "typical hypervisor cmdline",
			cmdlineContent: "console=tty0 rw consoleblank=0 iommu=pt intel_iommu=on security=apparmor systemd.gpt_auto=0 nowatchdog modprobe.blacklist=iTCO_wdt hugepagesz=2MB hugepages=1971167\n",
			expectedParams: "console=tty0 rw consoleblank=0 iommu=pt intel_iommu=on security=apparmor systemd.gpt_auto=0 nowatchdog modprobe.blacklist=iTCO_wdt hugepagesz=2MB hugepages=1971167",
			expectError:    false,
		},
		{
			name:           "minimal cmdline",
			cmdlineContent: "root=/dev/sda1 ro\n",
			expectedParams: "root=/dev/sda1 ro",
			expectError:    false,
		},
		{
			name:           "empty cmdline",
			cmdlineContent: "\n",
			expectedParams: "",
			expectError:    false,
		},
		{
			name:           "cmdline without trailing newline",
			cmdlineContent: "param1 param2",
			expectedParams: "param1 param2",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test cmdline content
			tmpDir := t.TempDir()
			cmdlinePath := filepath.Join(tmpDir, "cmdline")
			err := os.WriteFile(cmdlinePath, []byte(tt.cmdlineContent), 0644)
			require.NoError(t, err)

			// Create reader with custom path
			reader := NewSystemReaderWithPath(cmdlinePath)
			params, err := reader.ReadParameters()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedParams, params.CommandLine)
		})
	}
}

func TestReadParameters_FileNotFound(t *testing.T) {
	reader := NewSystemReaderWithPath("/nonexistent/path/cmdline")
	params, err := reader.ReadParameters()

	assert.Error(t, err)
	assert.Nil(t, params)
}

func TestNewSystemReader(t *testing.T) {
	reader := NewSystemReader()
	assert.Equal(t, DefaultCmdlinePath, reader.cmdlinePath)
}

package kcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canonical/rt-conf/src/model"
)

func TestProcessFile(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "grub")

		grub := model.Grub{
			CustomGrubFilePath: filePath,
			Cmdline:            "isolcpus=1-3 nohz=on",
		}

		err := processFile(grub)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		expected := `GRUB_CMDLINE_LINUX_DEFAULT="isolcpus=1-3 nohz=on"`
		if string(content) != expected {
			t.Errorf("expected content %q, got %q", expected, string(content))
		}
	})

	t.Run("FailToWriteFile", func(t *testing.T) {
		// Try writing to a directory that doesn't exist
		badPath := filepath.Join("/nonexistent-dir", "grub")

		grub := model.Grub{
			CustomGrubFilePath: badPath,
			Cmdline:            "isolcpus=0",
		}

		err := processFile(grub)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "failed to write to") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func setupTempFile(t *testing.T, content string, idex int) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("tempfile-%d", idex))
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func TestParseGrubFileHappy(t *testing.T) {
	var testCases = []struct {
		content string
		params  map[string]string
	}{
		{
			content: `GRUB_HIDDEN_TIMEOUT_QUIET=true`,
			params: map[string]string{
				"GRUB_HIDDEN_TIMEOUT_QUIET": "true",
			},
		},
		{
			content: `GRUB_TIMEOUT=2`,
			params: map[string]string{
				"GRUB_TIMEOUT": "2",
			},
		},
		{
			content: `GRUB_CMDLINE_LINUX_DEFAULT_DEFAULT="rootfstype=ext4 quiet splash acpi_osi="`,
			params: map[string]string{
				"GRUB_CMDLINE_LINUX_DEFAULT_DEFAULT": "rootfstype=ext4 quiet splash acpi_osi=",
			},
		},
		{
			content: "GRUB_DEFAULT=0\n" +
				"#GRUB_HIDDEN_TIMEOUT=0\n" +
				"GRUB_HIDDEN_TIMEOUT_QUIET=true\n" +
				"GRUB_TIMEOUT=2\n" +
				"GRUB_DISTRIBUTOR=`lsb_release -i -s 2> /dev/null || echo Debian`\n" +
				"GRUB_CMDLINE_LINUX_DEFAULT_DEFAULT=\"rootfstype=ext4 quiet splash acpi_osi=\"\n" +
				"GRUB_CMDLINE_LINUX_DEFAULT=\"\"\n",

			params: map[string]string{
				"GRUB_DEFAULT":                       "0",
				"GRUB_HIDDEN_TIMEOUT_QUIET":          "true",
				"GRUB_TIMEOUT":                       "2",
				"GRUB_DISTRIBUTOR":                   "`lsb_release -i -s 2> /dev/null || echo Debian`",
				"GRUB_CMDLINE_LINUX_DEFAULT_DEFAULT": "rootfstype=ext4 quiet splash acpi_osi=",
				"GRUB_CMDLINE_LINUX_DEFAULT":         "",
			},
		},
	}
	for i, tc := range testCases {
		tmpFile := setupTempFile(t, tc.content, i)
		t.Cleanup(func() {
			os.Remove(tmpFile)
		})

		params, err := ParseDefaultGrubFile(tmpFile)
		if err != nil {
			t.Fatalf("Failed to parse grub file: %v", err)
		}
		for k, v := range params {
			vt, ok := tc.params[k]
			if !ok {
				t.Fatalf("Expected %s not found", k)
			}
			if v != vt {
				t.Fatalf("Expected %s=%s, got %s=%s", k, vt, k, v)
			}

		}
	}
}

func TestDuplicatedParams(t *testing.T) {
	var testCases = []struct {
		name    string
		cmdline string
		err     error
	}{
		{
			name:    "No duplicates",
			cmdline: "quiet splash foo",
			err:     nil,
		},
		{
			name:    "Single parameter",
			cmdline: "quiet",
			err:     nil,
		},
		{
			name:    "Duplicate boolean parameters",
			cmdline: "quiet splash quiet",
			err:     nil,
		},
		{
			name:    "Duplicate keys with different values",
			cmdline: "potato=mashed potato=salad",
			err:     errors.New("duplicated parameter:"),
		},
		{
			name:    "Duplicate key-value pairs",
			cmdline: "potato=pie potato=pie",
			err:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := duplicatedParams(tc.cmdline)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("Expected error %v, got nil", tc.err)
				}
				if !strings.Contains(err.Error(), tc.err.Error()) {
					t.Fatalf("Expected error %v, got %v", tc.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		})
	}
}

func TestUpdateGrub(t *testing.T) {
	tests := []struct {
		name         string
		grubContent  string
		kcmd         model.KernelCmdline
		expectErr    string
		expectOutput string
	}{
		{
			name:        "No params to inject",
			grubContent: ``,
			kcmd:        model.KernelCmdline{},
			expectErr:   "no parameters to inject",
		},
		{
			name:        "ParseDefaultGrubFile fails",
			grubContent: "", // file will be removed
			kcmd: model.KernelCmdline{
				IsolCPUs: "1-3",
			},
			expectErr: "failed to parse grub file",
		},
		{
			name:        "GRUB_CMDLINE_LINUX_DEFAULT missing",
			grubContent: `GRUB_TIMEOUT=5`,
			kcmd: model.KernelCmdline{
				IsolCPUs: "1-3",
			},
			expectOutput: "Detected bootloader: GRUB",
		},
		{
			name: "Duplicate params found",
			grubContent: `GRUB_CMDLINE_LINUX_DEFAULT="isolcpus=1-3 isolcpus=2-4"
`,
			kcmd: model.KernelCmdline{
				IsolCPUs: "2-4",
			},
			expectErr: "invalid existing parameters",
		},
		{
			name: "ProcessFile fails",
			grubContent: `GRUB_CMDLINE_LINUX_DEFAULT="isolcpus=1-3"
`,
			kcmd: model.KernelCmdline{
				Nohz: "on",
			},
			expectErr: "error updating",
		},
		{
			name: "Success",
			grubContent: `GRUB_CMDLINE_LINUX_DEFAULT="isolcpus=1-3"
`,
			kcmd: model.KernelCmdline{
				IsolCPUs: "1-3",
				Nohz:     "on",
			},
			expectOutput: "Detected bootloader: GRUB",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			grubDefaultPath := filepath.Join(tmpDir, "grub")
			cfgPath := filepath.Join(tmpDir, "rt-conf.cfg")

			// If grub content exists, write it
			if tc.grubContent != "" {
				if err := os.WriteFile(grubDefaultPath, []byte(tc.grubContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// If test needs Parse failure, remove the file
			if tc.name == "ParseDefaultGrubFile fails" {
				os.Remove(grubDefaultPath)
				os.Remove(cfgPath)
			}

			conf := &model.InternalConfig{
				Data: model.Config{
					KernelCmdline: tc.kcmd,
				},
				GrubCfg: model.Grub{
					GrubDefaultFilePath: grubDefaultPath,
					CustomGrubFilePath:  cfgPath,
				},
			}

			if strings.Contains(tc.name, "ProcessFile fails") {
				processFile = func(_ model.Grub) error {
					return fmt.Errorf("mock write failure")
				}
			} else {
				processFile = func(_ model.Grub) error {
					// simulate a successful file process
					return nil
				}
			}

			msgs, err := UpdateGrub(conf)
			if tc.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected error %q, got: %v", tc.expectErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				found := false
				for _, msg := range msgs {
					if strings.Contains(msg, tc.expectOutput) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected output to contain %q, got %v", tc.expectOutput, msgs)
				}
			}
		})
	}
}

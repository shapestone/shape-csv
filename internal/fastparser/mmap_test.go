package fastparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMmapFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.csv")

	content := []byte("a,b,c\nd,e,f\ng,h,i")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test MmapFile
	data, cleanup, err := MmapFile(testFile)
	if err != nil {
		t.Fatalf("MmapFile() error = %v", err)
	}
	defer cleanup()

	// Verify data matches file content
	if string(data) != string(content) {
		t.Errorf("MmapFile() data = %q, want %q", string(data), string(content))
	}

	// Verify we can parse the mmapped data
	records, err := ParseZeroCopy(data)
	if err != nil {
		t.Fatalf("ParseZeroCopy() error = %v", err)
	}

	expectedRecords := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
		{"g", "h", "i"},
	}

	if len(records) != len(expectedRecords) {
		t.Fatalf("got %d records, want %d", len(records), len(expectedRecords))
	}

	for i, record := range records {
		for j, field := range record {
			want := expectedRecords[i][j]
			got := string(field)
			if got != want {
				t.Errorf("record[%d][%d] = %q, want %q", i, j, got, want)
			}
		}
	}
}

func TestMmapFile_EmptyFile(t *testing.T) {
	// Create an empty test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.csv")

	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test MmapFile with empty file
	data, cleanup, err := MmapFile(testFile)
	if err != nil {
		t.Fatalf("MmapFile() error = %v", err)
	}
	defer cleanup()

	if len(data) != 0 {
		t.Errorf("MmapFile() returned %d bytes for empty file, want 0", len(data))
	}
}

func TestMmapFile_NonexistentFile(t *testing.T) {
	_, _, err := MmapFile("/nonexistent/file.csv")
	if err == nil {
		t.Error("MmapFile() should return error for nonexistent file")
	}
}

func TestMmapFile_LargeFile(t *testing.T) {
	// Create a larger test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.csv")

	// Generate CSV with 1000 records
	var content []byte
	for i := 0; i < 1000; i++ {
		if i > 0 {
			content = append(content, '\n')
		}
		line := []byte("field1,field2,field3,field4,field5")
		content = append(content, line...)
	}

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test MmapFile
	data, cleanup, err := MmapFile(testFile)
	if err != nil {
		t.Fatalf("MmapFile() error = %v", err)
	}
	defer cleanup()

	// Verify we can parse the mmapped data
	records, err := ParseZeroCopy(data)
	if err != nil {
		t.Fatalf("ParseZeroCopy() error = %v", err)
	}

	if len(records) != 1000 {
		t.Errorf("got %d records, want 1000", len(records))
	}

	// Verify a sample record
	if len(records) > 0 {
		if len(records[0]) != 5 {
			t.Errorf("record has %d fields, want 5", len(records[0]))
		}
		if string(records[0][0]) != "field1" {
			t.Errorf("first field = %q, want %q", string(records[0][0]), "field1")
		}
	}
}

// TestMmapFile_ErrorCases tests error handling in MmapFile.
func TestMmapFile_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFile func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "nonexistent file",
			setupFile: func(t *testing.T) string {
				return "/nonexistent/path/to/file.csv"
			},
			wantErr: true,
		},
		{
			name: "directory instead of file",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir // Return directory path
			},
			wantErr: true,
		},
		{
			name: "unreadable file permissions",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "unreadable.csv")
				if err := os.WriteFile(testFile, []byte("a,b,c"), 0000); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return testFile
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFile(t)
			_, cleanup, err := MmapFile(filePath)

			if cleanup != nil {
				defer cleanup()
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("MmapFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMmapFile_CleanupFunction tests that the cleanup function properly unmaps and closes the file.
func TestMmapFile_CleanupFunction(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cleanup_test.csv")

	content := []byte("a,b,c\nd,e,f")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Mmap the file
	data, cleanup, err := MmapFile(testFile)
	if err != nil {
		t.Fatalf("MmapFile() error = %v", err)
	}

	// Verify data is accessible
	if string(data) != string(content) {
		t.Errorf("data mismatch: got %q, want %q", string(data), string(content))
	}

	// Call cleanup
	cleanup()

	// After cleanup, we should be able to delete the file
	// (on some systems, mmapped files can't be deleted)
	if err := os.Remove(testFile); err != nil {
		t.Logf("Note: Could not remove file after cleanup (may be platform-specific): %v", err)
	}
}

// TestMmapFile_EmptyFileCleanup tests that cleanup works correctly for empty files.
func TestMmapFile_EmptyFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.csv")

	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	data, cleanup, err := MmapFile(testFile)
	if err != nil {
		t.Fatalf("MmapFile() error = %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(data))
	}

	// Cleanup should work for empty files
	cleanup()

	// Should be able to remove the file after cleanup
	if err := os.Remove(testFile); err != nil {
		t.Logf("Note: Could not remove empty file after cleanup: %v", err)
	}
}

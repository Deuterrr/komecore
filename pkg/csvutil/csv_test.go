package csvutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"komecore/pkg/csvutil"
)

func TestLoadCSV(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")

	content := "header1,header2\nval1,val2\nval3,val4\n"
	if err := os.WriteFile(csvPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test csv: %v", err)
	}

	records, err := csvutil.LoadCSV(csvPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	if records[0][0] != "header1" || records[1][1] != "val2" {
		t.Errorf("unexpected record contents: %v", records)
	}

	// Test non-existent file
	_, err = csvutil.LoadCSV(filepath.Join(tmpDir, "nonexistent.csv"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

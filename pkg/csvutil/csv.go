package csvutil

import (
	"encoding/csv"
	"os"
)

// LoadCSV reads and parses an entire CSV file at the specified path into a slice of string records.
func LoadCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return csv.NewReader(f).ReadAll()
}

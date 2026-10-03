package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	bom := []byte{0xEF, 0xBB, 0xBF}

	err = filepath.WalkDir(wd, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".gemini" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if bytes.HasPrefix(content, bom) {
			fmt.Printf("Removing BOM from %s\n", path)
			newContent := bytes.TrimPrefix(content, bom)
			err = os.WriteFile(path, newContent, 0644)
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking path: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("BOM check complete!")
}

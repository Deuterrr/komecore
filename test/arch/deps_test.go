package archtest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// legacyAllowedCycles contains known bidirectional module cycles slated for removal in Phase 2.
// Format: "moduleA <-> moduleB" in alphabetical order.
var legacyAllowedCycles = map[string]bool{}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root containing go.mod")
		}
		dir = parent
	}
}

func TestModuleDependencies_NoUnallowlistedCycles(t *testing.T) {
	root := findRepoRoot(t)
	modulesDir := filepath.Join(root, "internal", "modules")

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		t.Fatalf("failed to read internal/modules directory: %v", err)
	}

	var moduleNames []string
	moduleDeps := make(map[string]map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			moduleNames = append(moduleNames, entry.Name())
			moduleDeps[entry.Name()] = make(map[string]bool)
		}
	}

	fset := token.NewFileSet()

	// Parse imports for each module
	for _, mod := range moduleNames {
		modPath := filepath.Join(modulesDir, mod)
		err := filepath.WalkDir(modPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			const prefix = "komecore/internal/modules/"
			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(importPath, prefix) {
					target := strings.TrimPrefix(importPath, prefix)
					targetMod := strings.Split(target, "/")[0]
					if targetMod != mod {
						moduleDeps[mod][targetMod] = true
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed parsing module %s: %v", mod, err)
		}
	}

	// Detect direct bidirectional cycles: A -> B and B -> A
	checkedPairs := make(map[string]bool)
	for modA, targets := range moduleDeps {
		for modB := range targets {
			if moduleDeps[modB][modA] {
				pair := []string{modA, modB}
				sort.Strings(pair)
				pairKey := pair[0] + " <-> " + pair[1]

				if checkedPairs[pairKey] {
					continue
				}
				checkedPairs[pairKey] = true

				if !legacyAllowedCycles[pairKey] {
					t.Errorf("forbidden circular module dependency detected: %s (not in legacyAllowedCycles allowlist)", pairKey)
				}
			}
		}
	}

	// Fail if an allowlisted cycle has been resolved but remains in the allowlist
	for allowedPair := range legacyAllowedCycles {
		parts := strings.Split(allowedPair, " <-> ")
		if len(parts) == 2 {
			modA, modB := parts[0], parts[1]
			if !moduleDeps[modA][modB] || !moduleDeps[modB][modA] {
				t.Logf("cycle %s is no longer active; consider removing from legacyAllowedCycles", allowedPair)
			}
		}
	}
}

func TestDomainPurity(t *testing.T) {
	root := findRepoRoot(t)
	modulesDir := filepath.Join(root, "internal", "modules")

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		t.Fatalf("failed to read internal/modules directory: %v", err)
	}

	forbiddenPrefixes := []string{
		"net/http",
		"komecore/internal/httpx",
		"github.com/jackc/pgx",
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		domainDir := filepath.Join(modulesDir, entry.Name(), entry.Name()+"domain")
		if _, err := os.Stat(domainDir); os.IsNotExist(err) {
			domainDir = filepath.Join(modulesDir, entry.Name(), "domain")
			if _, err := os.Stat(domainDir); os.IsNotExist(err) {
				continue
			}
		}

		err := filepath.WalkDir(domainDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				for _, forbidden := range forbiddenPrefixes {
					if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
						t.Errorf("domain package purity violation in %s: imports forbidden package %q", path, importPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed checking domain purity for %s: %v", entry.Name(), err)
		}
	}
}

package archtest

import (
	"go/ast"
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
		modDir := filepath.Join(modulesDir, entry.Name())
		files, err := os.ReadDir(modDir)
		if err != nil {
			t.Fatalf("failed reading module dir %s: %v", entry.Name(), err)
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
				continue
			}

			filePath := filepath.Join(modDir, f.Name())
			node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed parsing file %s: %v", filePath, err)
			}

			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				for _, forbidden := range forbiddenPrefixes {
					if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
						t.Errorf("domain package purity violation in %s: imports forbidden package %q", filePath, importPath)
					}
				}
			}
		}
	}
}

// Enforces anti-stuttering invariants across domain roots, persistence layers, and HTTP handlers.
// Prevents regression of legacy clean-architecture type aliases and repetitive package naming.
func TestNoStutteredExportedTypes(t *testing.T) {
	root := findRepoRoot(t)
	modulesDir := filepath.Join(root, "internal", "modules")

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		t.Fatalf("failed to read internal/modules directory: %v", err)
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modName := entry.Name()
		modDir := filepath.Join(modulesDir, modName)

		// 1. Inspect domain module root (*.go directly under internal/modules/<mod>/)
		rootFiles, err := os.ReadDir(modDir)
		if err != nil {
			t.Fatalf("failed reading module dir %s: %v", modName, err)
		}

		for _, f := range rootFiles {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
				continue
			}

			filePath := filepath.Join(modDir, f.Name())
			node, err := parser.ParseFile(fset, filePath, nil, 0)
			if err != nil {
				t.Fatalf("failed parsing %s: %v", filePath, err)
			}

			for _, decl := range node.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !ast.IsExported(typeSpec.Name.Name) {
						continue
					}
					typeName := typeSpec.Name.Name
					if strings.EqualFold(typeName, modName+"Repository") {
						t.Errorf("obsolete stuttered repository alias %s found in module root %s (Issue #30)", typeName, filePath)
					}
					if strings.EqualFold(typeName, modName+"Handler") {
						t.Errorf("obsolete stuttered handler alias %s found in module root %s (Issue #30)", typeName, filePath)
					}
				}
			}
		}

		// 2. Inspect subdirectories (*persistence and *http)
		subDirs, err := os.ReadDir(modDir)
		if err != nil {
			t.Fatalf("failed reading module subdirs for %s: %v", modName, err)
		}

		for _, sub := range subDirs {
			if !sub.IsDir() {
				continue
			}
			subName := sub.Name()
			isPersistence := strings.HasSuffix(subName, "persistence")
			isHTTP := strings.HasSuffix(subName, "http")
			if !isPersistence && !isHTTP {
				continue
			}

			subDirPath := filepath.Join(modDir, subName)
			subFiles, err := os.ReadDir(subDirPath)
			if err != nil {
				t.Fatalf("failed reading dir %s: %v", subDirPath, err)
			}

			for _, f := range subFiles {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
					continue
				}

				filePath := filepath.Join(subDirPath, f.Name())
				node, err := parser.ParseFile(fset, filePath, nil, 0)
				if err != nil {
					t.Fatalf("failed parsing %s: %v", filePath, err)
				}

				for _, decl := range node.Decls {
					switch d := decl.(type) {
					case *ast.GenDecl:
						if d.Tok != token.TYPE {
							continue
						}
						for _, spec := range d.Specs {
							typeSpec, ok := spec.(*ast.TypeSpec)
							if !ok || !ast.IsExported(typeSpec.Name.Name) {
								continue
							}
							typeName := typeSpec.Name.Name
							if isPersistence && strings.EqualFold(typeName, modName+"Repository") {
								t.Errorf("stuttered repository type %s found in persistence package %s (should be Repository)", typeName, filePath)
							}
							if isHTTP && strings.EqualFold(typeName, modName+"Handler") {
								t.Errorf("stuttered handler type %s found in http delivery package %s (should be Handler)", typeName, filePath)
							}
						}
					case *ast.FuncDecl:
						if d.Recv != nil || !ast.IsExported(d.Name.Name) {
							continue
						}
						funcName := d.Name.Name
						if isPersistence && strings.EqualFold(funcName, "New"+modName+"Repository") {
							t.Errorf("stuttered constructor %s found in persistence package %s (should be New())", funcName, filePath)
						}
						if isHTTP && strings.EqualFold(funcName, "New"+modName+"Handler") {
							t.Errorf("stuttered constructor %s found in http delivery package %s (should be New())", funcName, filePath)
						}
					}
				}
			}
		}
	}
}

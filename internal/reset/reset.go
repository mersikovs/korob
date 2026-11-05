// Package reset provides static analysis tools to discover Go structs
// marked for automatic reset logic and generation reset.gen.go  with this logic.
package reset

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CleanMode specifies the cleanup strategy for existing files during generation.
type CleanMode string

const (
	// CleanDelete indicates that existing files should be automatically removed.
	CleanDelete CleanMode = "delete"

	// CleanWarn indicates that a warning should be logged for existing files without removing them.
	CleanWarn CleanMode = "warn"
)

// ErrStaleFound signal about old reset.gen.go file.
var ErrStaleFound = errors.New("stale generated files found")

// Run scans the Go source tree rooted at root, aggregates resettable types
// per package, and generates a "reset.gen.go" file.
func Run(root string, cleanMode CleanMode) error {
	var (
		errs     []error
		anyStale bool
	)

	scanErr := scanPackages(root, func(dir string, files []string) error {
		var allTargetsInPkg []ResetTarget
		var pkgName string
		var parseFailed bool

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}

			targets, err := FindResetTargets(file)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: parse failed: %w", file, err))
				parseFailed = true
				break
			}

			if len(targets) > 0 {
				allTargetsInPkg = append(allTargetsInPkg, targets...)

				if pkgName == "" {
					pkgName = targets[0].PkgName
				}
			}
		}

		if parseFailed {
			return nil
		}

		if len(allTargetsInPkg) == 0 {
			err := handleStale(dir, cleanMode)
			if errors.Is(err, ErrStaleFound) {
				anyStale = true
			} else if err != nil {
				errs = append(errs, err)
			}
			return nil
		}

		fmt.Printf("Package %s: find %d structs\n", pkgName, len(allTargetsInPkg))
		if err := GenerateResetFileForPackage(dir, pkgName, allTargetsInPkg); err != nil {
			errs = append(errs, fmt.Errorf("%s: generate failed: %w", dir, err))
		}

		return nil
	})

	if scanErr != nil {
		errs = append(errs, fmt.Errorf("scan %s: %w", root, scanErr))
	}
	if anyStale {
		errs = append(errs, ErrStaleFound)
	}

	return errors.Join(errs...)
}

func handleStale(dir string, mode CleanMode) error {
	path := filepath.Join(dir, GeneratedFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%s: read failed: %w", path, err)
	}

	if !hasGeneratedMarker(data) {
		return fmt.Errorf("%s: exists but has no generated marker, refusing to delete", path)
	}

	switch mode {
	case CleanDelete:
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("%s: remove failed: %w", path, err)
		}
		return nil
	case CleanWarn:
		fmt.Fprintf(os.Stderr,
			"stale generated file %s: no types with directive, can be removed (use -clean=delete)\n", path)
		return ErrStaleFound
	}
	return fmt.Errorf("unknown clean mode %q", mode)
}

// ParseCleanMode parses the string s into a CleanMode.
func ParseCleanMode(s string) (CleanMode, error) {
	switch CleanMode(s) {
	case CleanDelete:
		return CleanDelete, nil
	case CleanWarn:
		return CleanWarn, nil
	case "":
		return "", fmt.Errorf("флаг -clean обязателен (delete|warn)")
	default:
		return "", fmt.Errorf("неизвестное значение -clean=%q (допустимо: delete|warn)", s)
	}
}

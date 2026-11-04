// Package reset provides static analysis tools to discover Go structs
// marked for automatic reset logic and generation reset.gen.go  with this logic.
package reset

import (
	"fmt"
	"strings"
)

// Run scans the Go source tree rooted at root, aggregates resettable types
// per package, and generates a "reset.gen.go" file.
func Run(root string) error {
	return scanPackages(root, func(dir string, files []string) error {
		var allTargetsInPkg []ResetTarget
		var pkgName string

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}

			targets, err := FindResetTargets(file)
			if err != nil {
				return fmt.Errorf("parse %s: %w", file, err)
			}

			if len(targets) > 0 {
				allTargetsInPkg = append(allTargetsInPkg, targets...)

				if pkgName == "" {
					pkgName = targets[0].PkgName
				}
			}
		}

		if len(allTargetsInPkg) > 0 {
			fmt.Printf("Package %s: find %d structs\n", pkgName, len(allTargetsInPkg))

			if err := GenerateResetFileForPackage(dir, pkgName, allTargetsInPkg); err != nil {
				return fmt.Errorf("package generate %s: %w", dir, err)
			}
		}

		return nil
	})
}

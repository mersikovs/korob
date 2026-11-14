package reset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// scanPackages it sequentially traverses the directories.
func scanPackages(root string, visit func(dir string, files []string) error) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("scanPackages: abs path error: %w", err)
	}

	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("scanPackages: root %q: %w", root, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("scanPackages: stat %q: %w", abs, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scanPackages: %q not dir", abs)
	}

	byDir := make(map[string][]string)

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("scanPackages: walkdir %s: %w", path, walkErr)
		}

		if d.IsDir() {
			if path != abs {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" {
					return fs.SkipDir
				}
			}
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		parent := filepath.Dir(path)
		byDir[parent] = append(byDir[parent], path)
		return nil
	})

	if err != nil {
		return err
	}

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)

	for _, dir := range dirs {
		files := byDir[dir]
		slices.Sort(files)
		if err := visit(dir, files); err != nil {
			return fmt.Errorf("scanPackages: package %s: %w", dir, err)
		}
	}

	return nil
}

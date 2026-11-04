package reset

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type scanCall struct {
	dir   string
	files []string
}

func TestScanPackages(t *testing.T) {
	t.Parallel()

	errVisit := errors.New("visit failed")

	tests := []struct {
		name         string
		setup        func(t *testing.T, root string)
		rootOverride func(t *testing.T, root string) string
		visitErr     error

		wantCalls []scanCall
		wantErr   string
		wantErrIs error
	}{

		{
			name:      "empty_root_no_visits",
			wantCalls: nil,
		},
		{
			name: "single_file",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "multiple_files_same_dir",
			setup: func(t *testing.T, root string) {

				writeGo(t, root, "b.go")
				writeGo(t, root, "a.go")
			},
			wantCalls: []scanCall{{".", []string{"a.go", "b.go"}}},
		},
		{
			name: "interleaved_files_and_dirs",

			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, "m/x.go")
				writeGo(t, root, "z.go")
			},
			wantCalls: []scanCall{
				{".", []string{"a.go", "z.go"}},
				{"m", []string{"m/x.go"}},
			},
		},
		{
			name: "nested_dirs",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, "sub/b.go")
				writeGo(t, root, "sub/deep/c.go")
			},
			wantCalls: []scanCall{
				{".", []string{"a.go"}},
				{"sub", []string{"sub/b.go"}},
				{"sub/deep", []string{"sub/deep/c.go"}},
			},
		},
		{
			name: "siblings",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "x/a.go")
				writeGo(t, root, "y/b.go")
			},
			wantCalls: []scanCall{
				{"x", []string{"x/a.go"}},
				{"y", []string{"y/b.go"}},
			},
		},

		{
			name: "skip_vendor",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, "vendor/v.go")
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "skip_testdata",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, "testdata/t.go")
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "skip_hidden_dir",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, ".git/h.go")
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "skip_non_go_files",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeTempFile(t, root, "readme.txt", "not go")
				writeTempFile(t, root, "config.yaml", "x: 1")
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "underscore_dir_not_skipped", // ⚠️ known behavior: _build/ не скипается
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				writeGo(t, root, "_build/b.go")
			},
			wantCalls: []scanCall{
				{".", []string{"a.go"}},
				{"_build", []string{"_build/b.go"}},
			},
		},
		{
			name: "skip_symlink_file",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				if err := os.Symlink(
					filepath.Join(root, "a.go"),
					filepath.Join(root, "link.go"),
				); err != nil {
					t.Skipf("symlink unsupported: %v", err)
				}
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "empty_subdirs_not_visited",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
				if err := os.Mkdir(filepath.Join(root, "empty"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "only_hidden_subdir",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, ".hidden/x.go")
			},
			wantCalls: nil,
		},

		{
			name: "root_is_symlink",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
			},
			rootOverride: func(t *testing.T, root string) string {
				link := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(root, link); err != nil {
					t.Skipf("symlink unsupported: %v", err)
				}
				return link
			},
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
		{
			name: "missing_root",
			rootOverride: func(t *testing.T, root string) string {
				return filepath.Join(root, "nonexistent")
			},
			wantErr:   "scanPackages: root",
			wantErrIs: fs.ErrNotExist,
			wantCalls: nil,
		},
		{
			name: "root_is_file_not_dir",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
			},
			rootOverride: func(t *testing.T, root string) string {
				return filepath.Join(root, "a.go")
			},
			wantErr:   "not dir",
			wantCalls: nil,
		},

		{
			name: "visit_error_propagates",
			setup: func(t *testing.T, root string) {
				writeGo(t, root, "a.go")
			},
			visitErr:  errVisit,
			wantErr:   "scanPackages: package",
			wantErrIs: errVisit,
			wantCalls: []scanCall{{".", []string{"a.go"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, root)
			}
			callRoot := root
			if tt.rootOverride != nil {
				callRoot = tt.rootOverride(t, root)
			}

			evalRoot, err := filepath.EvalSymlinks(callRoot)
			if err != nil {
				evalRoot = callRoot
			}

			var calls []scanCall
			err = scanPackages(callRoot, func(dir string, files []string) error {
				relDir, _ := filepath.Rel(evalRoot, dir)
				relFiles := make([]string, 0, len(files))
				for _, f := range files {
					rel, _ := filepath.Rel(evalRoot, f)
					relFiles = append(relFiles, filepath.ToSlash(rel))
				}
				calls = append(calls, scanCall{filepath.ToSlash(relDir), relFiles})
				if tt.visitErr != nil {
					return tt.visitErr
				}
				return nil
			})

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErrIs, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Errorf("calls mismatch:\n got: %+v\nwant: %+v", calls, tt.wantCalls)
			}
		})
	}
}

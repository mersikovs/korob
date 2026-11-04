package reset

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

type fileState int

const (
	fileAbsent fileState = iota
	fileExists
	fileEqualsExisting
)

func TestGenerateResetFileForPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		existing      []byte
		useMissingDir bool
		pkg           string
		targets       []ResetTarget

		wantErrContains string
		wantErrIs       error
		wantFile        fileState
		wantValidGo     bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:     "empty_targets_no_op",
			pkg:      "mypkg",
			targets:  nil,
			wantFile: fileAbsent,
		},
		{
			name:     "empty_targets_preserves_existing_file",
			existing: []byte("package mypkg\n\n// sentinel: file from previous run\n"),
			pkg:      "mypkg",
			targets:  []ResetTarget{},
			wantFile: fileEqualsExisting,
		},
		{
			name: "happy_path",
			pkg:  "mypkg",
			targets: []ResetTarget{
				target("Profile", mkField("Name", "string")),
				target("Session", mkField("ID", "int")),
			},
			wantFile:     fileExists,
			wantValidGo:  true,
			wantContains: []string{"package mypkg", "Profile", "Session"},
		},
		{
			name:     "overwrites_existing_file",
			existing: []byte("package old\n\n// stale from previous run\n"),
			pkg:      "mypkg",
			targets: []ResetTarget{
				target("Profile", mkField("Name", "string")),
				target("Session", mkField("ID", "int")),
			},
			wantFile:        fileExists,
			wantValidGo:     true,
			wantContains:    []string{"package mypkg", "Profile", "Session"},
			wantNotContains: []string{"stale from previous run"},
		},
		{
			name:          "write_error_missing_dir",
			useMissingDir: true,
			pkg:           "mypkg",
			targets: []ResetTarget{
				target("Profile", mkField("Name", "string")),
			},
			wantErrContains: "write file",
			wantErrIs:       fs.ErrNotExist,
			wantFile:        fileAbsent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			dir := base
			if tt.useMissingDir {
				dir = filepath.Join(base, "missing")
			}
			path := filepath.Join(dir, "reset.gen.go")

			if tt.existing != nil {
				if err := os.WriteFile(path, tt.existing, 0600); err != nil {
					t.Fatal(err)
				}
			}

			err := GenerateResetFileForPackage(dir, tt.pkg, tt.targets)

			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err, tt.wantErrContains)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErrIs, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			switch tt.wantFile {
			case fileAbsent:
				if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("file must not exist; stat err = %v", statErr)
				}
				return

			case fileEqualsExisting:
				got, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatalf("read file: %v", readErr)
				}
				if !bytes.Equal(got, tt.existing) {
					t.Errorf("file must equal existing content:\n got: %q\nwant: %q",
						got, tt.existing)
				}
				return

			case fileExists:

			}

			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read file: %v", readErr)
			}
			if tt.wantValidGo {
				assertValidGo(t, got)
			}
			for _, s := range tt.wantContains {
				if !bytes.Contains(got, []byte(s)) {
					t.Errorf("file missing %q; got:\n%s", s, got)
				}
			}
			for _, s := range tt.wantNotContains {
				if bytes.Contains(got, []byte(s)) {
					t.Errorf("file must not contain %q; got:\n%s", s, got)
				}
			}
			for _, tg := range tt.targets {
				for _, f := range tg.Fields {
					assertContainsReset(t, got, f)
				}
			}
		})
	}
}

func TestResetTemplate_Parses(t *testing.T) {
	t.Parallel()

	_, err := template.New("reset").Funcs(template.FuncMap{
		"resetExpr": getResetExpression,
	}).Parse(resetTemplate)
	if err != nil {
		t.Fatalf("resetTemplate is not a valid template: %v", err)
	}
}

func TestRenderResetCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pkg     string
		targets []ResetTarget
	}{
		{
			name: "nil_targets", pkg: "mypkg", targets: nil,
		},
		{
			name: "empty_targets", pkg: "mypkg", targets: []ResetTarget{},
		},
		{
			name: "single_target_single_field", pkg: "mypkg",
			targets: []ResetTarget{
				target("Profile", mkField("Name", "string")),
			},
		},
		{
			name: "single_target_many_fields", pkg: "mypkg",
			targets: []ResetTarget{
				target("Profile",
					mkField("Name", "string"),
					mkField("Age", "int"),
					mkField("Tags", "[]int"),
					mkField("Meta", "map[string]any"),
					mkField("Ptr", "*int"),
				),
			},
		},
		{
			name: "multiple_targets", pkg: "p",
			targets: []ResetTarget{
				target("A", mkField("X", "int")),
				target("B", mkField("Y", "string")),
				target("C", mkField("Z", "float64")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, err := renderResetCode(tt.pkg, tt.targets)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertPackageClause(t, out, tt.pkg)
			assertValidGo(t, out)

			for _, tg := range tt.targets {
				if !bytes.Contains(out, []byte(tg.TypeName)) {
					t.Errorf("output missing type %q; got:\n%s", tg.TypeName, out)
				}
				for _, f := range tg.Fields {
					assertContainsReset(t, out, f)
				}
			}
		})
	}
}

func TestRenderResetCode_Deterministic(t *testing.T) {
	t.Parallel()

	targets := []ResetTarget{
		target("T",
			mkField("A", "int"),
			mkField("B", "string"),
		),
	}

	a, err := renderResetCode("p", targets)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	b, err := renderResetCode("p", targets)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("non-deterministic output:\n--- A ---\n%s\n--- B ---\n%s", a, b)
	}
}

func mkField(name, typ string) Field { return Field{Name: name, TypeName: typ} }
func fieldType(f Field) string       { return f.TypeName }

func target(typeName string, fields ...Field) ResetTarget {
	return ResetTarget{
		PkgName:  "mypkg",
		TypeName: typeName,
		Fields:   fields,
	}
}

func assertPackageClause(t *testing.T, out []byte, pkg string) {
	t.Helper()
	want := "package " + pkg
	if !bytes.Contains(out, []byte(want)) {
		t.Errorf("output missing %q; got:\n%s", want, out)
	}
}

func assertValidGo(t *testing.T, out []byte) {
	t.Helper()
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "generated.go", out, parser.AllErrors); err != nil {
		t.Errorf("output is not valid Go: %v\nsource:\n%s", err, out)
	}
}

func assertContainsReset(t *testing.T, out []byte, f Field) {
	t.Helper()
	for _, recv := range []string{"r.", "v.", "x.", "recv.", "self."} {
		want := []byte(getResetExpression(recv, f.Name, fieldType(f)))
		if bytes.Contains(out, want) {
			return
		}
	}
	t.Errorf("output has no reset expression for field %s (%s); got:\n%s",
		f.Name, fieldType(f), out)
}

func mustFormat(t *testing.T, src string) []byte {
	t.Helper()
	out, err := format.Source([]byte(src))
	if err != nil {
		t.Fatalf("test setup: format.Source(%q): %v", src, err)
	}
	return out
}

func TestWriteFormattedFile(t *testing.T) {
	t.Parallel()

	sentinel := []byte("package old\n\n// sentinel\n")

	tests := []struct {
		name      string
		pathFn    func(dir string) string
		prepare   func(t *testing.T, path string)
		raw       []byte
		wantErr   string
		wantErrIs error
		wantFile  []byte
	}{
		{
			name:     "valid_unformatted_applies_gofmt",
			raw:      []byte("package p\nfunc  f( ){\n}\n"),
			wantFile: mustFormat(t, "package p\nfunc  f( ){\n}\n"),
		},
		{
			name:     "valid_already_formatted_idempotent",
			raw:      []byte("package p\n\nfunc f() {}\n"),
			wantFile: mustFormat(t, "package p\n\nfunc f() {}\n"),
		},
		{
			name: "overwrites_existing_file",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, sentinel, 0600); err != nil {
					t.Fatal(err)
				}
			},
			raw:      []byte("package p\nfunc f(){}\n"),
			wantFile: mustFormat(t, "package p\nfunc f(){}\n"),
		},
		{
			name:    "invalid_syntax_returns_error_with_raw_code",
			raw:     []byte("this is not Go"),
			wantErr: "format source",
		},
		{
			name: "invalid_syntax_preserves_existing_file",
			prepare: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, sentinel, 0600); err != nil {
					t.Fatal(err)
				}
			},
			raw:      []byte("this is not Go"),
			wantErr:  "format source",
			wantFile: sentinel,
		},
		{
			name:    "not_valid",
			raw:     []byte("package p\nvar x = "),
			wantErr: "format source",
		},
		{
			name:      "write_error_missing_parent_dir",
			pathFn:    func(dir string) string { return filepath.Join(dir, "missing", "out.go") },
			raw:       []byte("package p\nfunc f(){}\n"),
			wantErr:   "write file",
			wantErrIs: fs.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "out.go")
			if tt.pathFn != nil {
				path = tt.pathFn(dir)
			}
			if tt.prepare != nil {
				tt.prepare(t, path)
			}

			err := writeFormattedFile(path, tt.raw)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}

				if strings.Contains(tt.wantErr, "format") && len(tt.raw) > 0 &&
					!strings.Contains(err.Error(), string(tt.raw)) {
					t.Errorf("format error should embed raw code; err=%q", err)
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("errors.Is(err, %v) = false; err=%v", tt.wantErrIs, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantFile == nil {
				if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
					t.Errorf("file %q must not exist after failed call; stat err = %v", path, statErr)
				}
				return
			}

			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read %q: %v", path, readErr)
			}
			if !bytes.Equal(got, tt.wantFile) {
				t.Errorf("file content mismatch:\n got: %q\nwant: %q", got, tt.wantFile)
			}
		})
	}
}

func TestHandlePointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		receiver  string
		fieldName string
		baseType  string
		want      string
	}{

		{
			name: "resettable_struct", receiver: "s.", fieldName: "P", baseType: "MyStruct",
			want: fmt.Sprintf(`if r, ok := any(s.P).(resettable); ok {
				r.Reset()
			} else if s.P != nil {
				*s.P = %s
			}`, getZeroValue("MyStruct")),
		},
		{
			name: "slice_fallback", receiver: "s.", fieldName: "P", baseType: "[]int",
			want: fmt.Sprintf(`if r, ok := any(s.P).(resettable); ok {
				r.Reset()
			} else if s.P != nil {
				*s.P = %s
			}`, getZeroValue("[]int")),
		},
		{
			name: "map_fallback", receiver: "s.", fieldName: "P", baseType: "map[string]int",
			want: fmt.Sprintf(`if r, ok := any(s.P).(resettable); ok {
				r.Reset()
			} else if s.P != nil {
				*s.P = %s
			}`, getZeroValue("map[string]int")),
		},

		{
			name: "primitive_int", receiver: "s.", fieldName: "P", baseType: "int",
			want: fmt.Sprintf(`if s.P != nil {
				*s.P = %s
			}`, getZeroValue("int")),
		},
		{
			name: "primitive_string", receiver: "s.", fieldName: "P", baseType: "string",
			want: fmt.Sprintf(`if s.P != nil {
				*s.P = %s
			}`, getZeroValue("string")),
		},
		{
			name: "primitive_any", receiver: "s.", fieldName: "P", baseType: "any",
			want: fmt.Sprintf(`if s.P != nil {
				*s.P = %s
			}`, getZeroValue("any")),
		},
		{
			name: "qualified_time", receiver: "p.", fieldName: "T", baseType: "time.Time",
			want: fmt.Sprintf(`if p.T != nil {
				*p.T = %s
			}`, getZeroValue("time.Time")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatGenerated(t, handlePointer(tt.receiver, tt.fieldName, tt.baseType))
			want := formatGenerated(t, tt.want)

			if got != want {
				t.Errorf("handlePointer(%q, %q, %q) mismatch:\n\n got:\n%s\n\nwant:\n%s",
					tt.receiver, tt.fieldName, tt.baseType, got, want)
			}
		})
	}
}

func formatGenerated(t *testing.T, src string) string {
	t.Helper()
	wrapped := "package p\nfunc _() {\n" + src + "\n}\n"
	out, err := format.Source([]byte(wrapped))
	if err != nil {
		t.Fatalf("generated code does not parse: %v\nsource:\n%s", err, wrapped)
	}
	return string(out)
}

func TestGetResetExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		receiver  string
		fieldName string
		typeName  string
		want      string
	}{
		{
			name: "map_string_int", receiver: "s.", fieldName: "M", typeName: "map[string]int",
			want: `clear(s.M)`,
		},
		{
			name: "map_string_any", receiver: "s.", fieldName: "Cache", typeName: "map[string]any",
			want: `clear(s.Cache)`,
		},

		{
			name: "ptr_int", receiver: "s.", fieldName: "P", typeName: "*int",
			want: fmt.Sprintf("if s.P != nil {\n\t*s.P = %s\n}", getZeroValue("int")),
		},
		{
			name: "ptr_qualified", receiver: "s.", fieldName: "P", typeName: "*pkg.Type",
			want: fmt.Sprintf("if s.P != nil {\n\t*s.P = %s\n}", getZeroValue("pkg.Type")),
		},
		{
			name: "ptr_struct", receiver: "s.", fieldName: "P", typeName: "*MyStruct",
			want: fmt.Sprintf("if r, ok := any(s.P).(resettable); ok {\nr.Reset()\n} else if s.P != nil {\n*s.P = %s\n}", getZeroValue("MyStruct")),
		},
		{
			name: "ptr_to_slice", receiver: "s.", fieldName: "P", typeName: "*[]int",
			want: fmt.Sprintf("if r, ok := any(s.P).(resettable); ok {\nr.Reset()\n} else if s.P != nil {\n*s.P = %s\n}", getZeroValue("[]int")),
		},
		{
			name: "ptr_to_map", receiver: "s.", fieldName: "P", typeName: "*map[string]int",
			want: fmt.Sprintf("if r, ok := any(s.P).(resettable); ok {\nr.Reset()\n} else if s.P != nil {\n*s.P = %s\n}", getZeroValue("map[string]int")),
		},

		{
			name: "slice_int", receiver: "s.", fieldName: "IDs", typeName: "[]int",
			want: `s.IDs = nil`,
		},
		{
			name: "slice_struct", receiver: "s.", fieldName: "Addresses", typeName: "[]Address",
			want: `for i := range s.Addresses {
				if r, ok := any(&s.Addresses[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},
		{
			name: "slice_ptr", receiver: "s.", fieldName: "Profiles", typeName: "[]*Profile",
			want: `for i := range s.Profiles {
				if r, ok := any(s.Profiles[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},

		{
			name: "struct_resettable", receiver: "s.", fieldName: "A", typeName: "Address",
			want: fmt.Sprintf(`if r, ok := any(&s.A).(resettable); ok {
				r.Reset()
			} else {
				s.A = %s
			}`, getZeroValue("Address")),
		},

		{
			name: "primitive_int", receiver: "s.", fieldName: "N", typeName: "int",
			want: fmt.Sprintf("s.N = %s", getZeroValue("int")),
		},
		{
			name: "primitive_string", receiver: "s.", fieldName: "S", typeName: "string",
			want: fmt.Sprintf("s.S = %s", getZeroValue("string")),
		},
		{
			name: "qualified_time", receiver: "p.", fieldName: "T", typeName: "time.Time",
			want: fmt.Sprintf("p.T = %s", getZeroValue("time.Time")),
		},
		{
			name: "any", receiver: "s.", fieldName: "V", typeName: "any",
			want: fmt.Sprintf("s.V = %s", getZeroValue("any")),
		},
		{
			name: "error", receiver: "s.", fieldName: "E", typeName: "error",
			want: fmt.Sprintf("s.E = %s", getZeroValue("error")),
		},

		{
			name: "trim_map", receiver: "s.", fieldName: "M", typeName: "  map[string]int  ",
			want: `clear(s.M)`,
		},
		{
			name: "trim_ptr", receiver: "s.", fieldName: "P", typeName: "  *int  ",
			want: fmt.Sprintf("if s.P != nil {\n\t*s.P = %s\n}", getZeroValue("int")),
		},
		{
			name: "trim_slice", receiver: "s.", fieldName: "IDs", typeName: "  []int  ",
			want: `s.IDs = nil`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatGenerated(t, getResetExpression(tt.receiver, tt.fieldName, tt.typeName))
			want := formatGenerated(t, tt.want)

			if got != want {
				t.Errorf("getResetExpression(%q, %q, %q) mismatch:\n\n got:\n%s\n\nwant:\n%s",
					tt.receiver, tt.fieldName, tt.typeName, got, want)
			}
		})
	}
}

func TestHandleSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		receiver  string
		fieldName string
		elemType  string
		want      string
	}{
		{
			name: "nil_primitive", receiver: "s.", fieldName: "IDs", elemType: "int",
			want: `s.IDs = nil`,
		},
		{
			name: "nil_string", receiver: "s.", fieldName: "Tags", elemType: "string",
			want: `s.Tags = nil`,
		},
		{
			name: "nil_qualified", receiver: "p.", fieldName: "Times", elemType: "time.Time",
			want: `p.Times = nil`,
		},

		{
			name: "nil_pointer_primitive", receiver: "s.", fieldName: "Ptrs", elemType: "*int",
			want: `s.Ptrs = nil`,
		},
		{
			name: "nil_pointer_qualified", receiver: "s.", fieldName: "Refs", elemType: "*pkg.Type",
			want: `s.Refs = nil`,
		},

		{
			name: "value_struct", receiver: "s.", fieldName: "Addresses", elemType: "Address",
			want: `for i := range s.Addresses {
				if r, ok := any(&s.Addresses[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},
		{
			name: "value_slice_elem", receiver: "s.", fieldName: "Matrix", elemType: "[]int",
			want: `for i := range s.Matrix {
				if r, ok := any(&s.Matrix[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},
		{
			name: "value_map_elem", receiver: "s.", fieldName: "Maps", elemType: "map[string]int",
			want: `for i := range s.Maps {
				if r, ok := any(&s.Maps[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},

		{
			name: "pointer_struct", receiver: "s.", fieldName: "Profiles", elemType: "*Profile",
			want: `for i := range s.Profiles {
				if r, ok := any(s.Profiles[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},
		{
			name: "pointer_other_receiver", receiver: "x.", fieldName: "Children", elemType: "*Node",
			want: `for i := range x.Children {
				if r, ok := any(x.Children[i]).(resettable); ok {
					r.Reset()
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatGenerated(t, handleSlice(tt.receiver, tt.fieldName, tt.elemType))
			want := formatGenerated(t, tt.want)

			if got != want {
				t.Errorf("handleSlice(%q, %q, %q) mismatch after gofmt:\n\n got:\n%s\n\nwant:\n%s",
					tt.receiver, tt.fieldName, tt.elemType, got, want)
			}
		})
	}
}

func TestMightBeResettable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseType string
		want     bool
	}{

		{"qualified", "pkg.Type", false},
		{"leading_dot", ".Type", false},
		{"trailing_dot", "Type.", false},

		{"composite_with_dot", "[]pkg.Type", false},

		{"bool", "bool", false},
		{"string", "string", false},
		{"int", "int", false},
		{"int8", "int8", false},
		{"int16", "int16", false},
		{"int32", "int32", false},
		{"int64", "int64", false},
		{"uint", "uint", false},
		{"uint8", "uint8", false},
		{"uint16", "uint16", false},
		{"uint32", "uint32", false},
		{"uint64", "uint64", false},
		{"uintptr", "uintptr", false},
		{"float32", "float32", false},
		{"float64", "float64", false},
		{"complex64", "complex64", false},
		{"complex128", "complex128", false},
		{"byte", "byte", false},
		{"rune", "rune", false},
		{"any", "any", false},
		{"error", "error", false},

		{"simple_ident", "MyStruct", true},
		{"slice", "[]int", true},
		{"map", "map[string]int", true},
		{"pointer", "*MyStruct", true},

		{"uppercase_primitive", "Int", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mightBeResettable(tt.baseType); got != tt.want {
				t.Errorf("mightBeResettable(%q) = %v, want %v",
					tt.baseType, got, tt.want)
			}
		})
	}
}

func TestGetZeroValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeName string
		want     string
	}{

		{name: "empty string", typeName: "", want: ""},
		{name: "only whitespace", typeName: "\t \n", want: ""},

		{name: "pointer to int", typeName: "*int", want: "nil"},
		{name: "double pointer", typeName: "**int", want: "nil"},

		{name: "chan bidirectional", typeName: "chan int", want: "nil"},
		{name: "chan send-only", typeName: "chan<- int", want: "nil"},
		{name: "chan recv-only", typeName: "<-chan int", want: "nil"},

		{name: "func", typeName: "func()", want: "nil"},

		{name: "interface{}", typeName: "interface{}", want: "nil"},
		{name: "any", typeName: "any", want: "nil"},
		{name: "error", typeName: "error", want: "nil"},
		{name: "custom interface", typeName: "interface{ Foo() }", want: "nil"},

		{name: "string", typeName: "string", want: `""`},
		{name: "bool", typeName: "bool", want: "false"},
		{name: "int", typeName: "int", want: "0"},
		{name: "byte", typeName: "byte", want: "0"},
		{name: "rune", typeName: "rune", want: "0"},

		{name: "named struct", typeName: "time.Time", want: "time.Time{}"},
		{name: "fixed array", typeName: "[3]int", want: "[3]int{}"},

		{name: "spaces around int", typeName: "   int   ", want: "0"},
		{name: "spaces around pointer", typeName: "  *int  ", want: "nil"},
		{name: "spaces around chan", typeName: "  chan int  ", want: "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getZeroValue(tt.typeName); got != tt.want {
				t.Errorf("getZeroValue(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

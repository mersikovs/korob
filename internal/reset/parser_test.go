package reset

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestFindResetTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		wantErr string // substring; "" not error
		want    []ResetTarget
	}{

		{
			name:    "parse_error",
			src:     "package p\nfunc broken( {\n",
			wantErr: "parse ",
		},

		{
			name: "empty_file",
			src:  "package p\n",
			want: nil,
		},
		{
			name: "type_without_directive",
			src: `package p

				type A struct{ X int }
				`,
			want: nil,
		},
		{
			name: "directive_on_func",
			src: `package p

				// generate:reset
				func f() {}
				`,
			want: nil,
		},
		{
			name: "directive_on_var",
			src: `package p

				// generate:reset
				var X = 1
				`,
			want: nil,
		},
		{
			name: "directive_on_non_struct",
			src: `package p

					// generate:reset
					type Alias = int
					`,
			want: nil,
		},
		{
			name: "directive_on_interface",
			src: `package p

				// generate:reset
				type Iface interface{ M() }
				`,
			want: nil,
		},
		{
			name: "wrong_directive_format",
			src: `package p

				//go:generate:reset
				type A struct{}
				`,
			want: nil,
		},
		{
			name: "wrong_directive_name",
			src: `package p

				// generate:other
				type A struct{}
				`,
			want: nil,
		},
		{
			name: "block_comment_directive",
			src: `package p

				/* generate:reset */
				type A struct{}
				`,
			want: nil,
		},
		{

			name: "directive_on_spec_not_group",
			src: `package p

				type (
					// generate:reset
					A struct{}
					B struct{}
				)
				`,
			want: nil,
		},

		{
			name: "single_target",
			src: `package p

				// generate:reset
				type Profile struct{ Name string }
				`,
			want: []ResetTarget{
				{
					PkgName:  "p",
					Line:     4,
					TypeName: "Profile",
					Fields:   []Field{{Name: "Name", TypeName: "string"}},
				},
			},
		},
		{
			name: "empty_struct",
			src: `package p

				// generate:reset
				type Empty struct{}
				`,
			want: []ResetTarget{
				{PkgName: "p", Line: 4, TypeName: "Empty", Fields: nil},
			},
		},
		{
			name: "fields_propagated",
			src: `package p

				// generate:reset
				type S struct {
					ID   int    ` + "`json:\"id\"`" + `
					Name string
				}
				`,
			want: []ResetTarget{
				{
					PkgName:  "p",
					Line:     4,
					TypeName: "S",
					Fields: []Field{
						{Name: "ID", TypeName: "int"},
						{Name: "Name", TypeName: "string"},
					},
				},
			},
		},

		{
			name: "two_separate_directives",
			src: `package p

				// generate:reset
				type A struct{ X int }

				// generate:reset
				type B struct{ Y string }
				`,
			want: []ResetTarget{
				{PkgName: "p", Line: 4, TypeName: "A", Fields: []Field{{Name: "X", TypeName: "int"}}},
				{PkgName: "p", Line: 7, TypeName: "B", Fields: []Field{{Name: "Y", TypeName: "string"}}},
			},
		},
		{
			name: "grouped_types_one_directive",
			src: `package p

				// generate:reset
				type (
					A struct{ X int }
					B struct{ Y string }
				)
				`,
			want: []ResetTarget{
				{PkgName: "p", Line: 5, TypeName: "A", Fields: []Field{{Name: "X", TypeName: "int"}}},
				{PkgName: "p", Line: 6, TypeName: "B", Fields: []Field{{Name: "Y", TypeName: "string"}}},
			},
		},
		{
			name: "grouped_mixed_struct_and_alias",
			src: `package p

				// generate:reset
				type (
					A struct{ X int }
					B = int
				)
				`,
			want: []ResetTarget{
				{PkgName: "p", Line: 5, TypeName: "A", Fields: []Field{{Name: "X", TypeName: "int"}}},
			},
		},
		{
			name: "directive_between_types",
			src: `package p

				type Before struct{ A int }

				// generate:reset
				type Middle struct{ B int }

				type After struct{ C int }
				`,
			want: []ResetTarget{
				{PkgName: "p", Line: 6, TypeName: "Middle", Fields: []Field{{Name: "B", TypeName: "int"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeTempGo(t, tt.src)

			got, err := FindResetTargets(path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("error must name the file %q; got: %v", path, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindResetTargets:\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestHasDirective(t *testing.T) {
	t.Parallel()

	commentGroup := func(texts ...string) *ast.CommentGroup {
		if len(texts) == 0 {
			return &ast.CommentGroup{}
		}
		comments := make([]*ast.Comment, len(texts))
		for i, txt := range texts {
			comments[i] = &ast.Comment{Text: txt}
		}
		return &ast.CommentGroup{List: comments}
	}

	tests := []struct {
		name   string
		cg     *ast.CommentGroup
		want   string
		expect bool
	}{

		{"nil_group", nil, "generate:reset", false},
		{"empty_list", commentGroup(), "generate:reset", false},

		{"exact_match", commentGroup("// generate:reset"), "generate:reset", true},

		{"trailing_space", commentGroup("// generate:reset "), "generate:reset", true},

		{"no_space_after_slashes", commentGroup("//generate:reset"), "generate:reset", false},
		{"go_generate_format", commentGroup("//go:generate:reset"), "generate:reset", false},

		{"double_space", commentGroup("//  generate:reset"), "generate:reset", false},
		{"tab_separator", commentGroup("//\tgenerate:reset"), "generate:reset", false},

		{"block_comment", commentGroup("/* generate:reset */"), "generate:reset", false},

		{"different_directive", commentGroup("// generate:other"), "generate:reset", false},
		{"plus_prefix", commentGroup("//+generate:reset"), "generate:reset", false},

		{"found_in_middle", commentGroup("// foo", "// generate:reset", "// bar"), "generate:reset", true},
		{"found_first", commentGroup("// generate:reset", "// foo"), "generate:reset", true},
		{"found_last", commentGroup("// foo", "// generate:reset"), "generate:reset", true},
		{"not_found_among_many", commentGroup("// foo", "// bar", "// baz"), "generate:reset", false},

		{"custom_want", commentGroup("// custom:directive"), "custom:directive", true},
		{"want_short", commentGroup("// foo"), "foo", true},

		{"empty_want_with_space", commentGroup("// "), "", false},
		{"empty_want_no_space", commentGroup("//"), "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasDirective(tt.cg, tt.want); got != tt.expect {
				t.Errorf("hasDirective = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestHasDirective_FromParser(t *testing.T) {
	t.Parallel()

	src := `package p

// generate:reset
type T struct{}

//  generate:reset
type A struct{}

//generate:reset
type B struct{}

// other:directive
type C struct{}

type D struct{}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	type decl struct {
		name string
		doc  *ast.CommentGroup
	}
	var decls []decl
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			decls = append(decls, decl{ts.Name.Name, gen.Doc})
		}
	}

	want := []bool{true, false, false, false, false}
	if len(decls) != len(want) {
		t.Fatalf("got %d types, want %d", len(decls), len(want))
	}
	for i, d := range decls {
		if got := hasDirective(d.doc, "generate:reset"); got != want[i] {
			t.Errorf("type %s: hasDirective = %v, want %v", d.name, got, want[i])
		}
	}
}

func TestExtractFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []Field
	}{

		{
			name: "empty_struct",
			src:  "package p\ntype T struct{}",
			want: nil,
		},

		{
			name: "single_field",
			src:  "package p\ntype T struct{ A int }",
			want: []Field{{Name: "A", TypeName: "int"}},
		},
		{
			name: "multi_fields_preserve_order",
			src:  "package p\ntype T struct{ A int; B string; C bool }",
			want: []Field{
				{Name: "A", TypeName: "int"},
				{Name: "B", TypeName: "string"},
				{Name: "C", TypeName: "bool"},
			},
		},
		{
			name: "underscore_name",
			src:  "package p\ntype T struct{ _ int }",
			want: []Field{{Name: "_", TypeName: "int"}},
		},

		{
			name: "field_with_tag",
			src:  "package p\ntype T struct{ A int `json:\"a\"` }",
			want: []Field{{Name: "A", TypeName: "int"}},
		},

		{
			name: "two_names_same_type",
			src:  "package p\ntype T struct{ A, B int }",
			want: []Field{
				{Name: "A", TypeName: "int"},
				{Name: "B", TypeName: "int"},
			},
		},
		{
			name: "two_names_with_tag",
			src:  "package p\ntype T struct{ A, B int `json:\"ab\"` }",
			want: []Field{
				{Name: "A", TypeName: "int"},
				{Name: "B", TypeName: "int"},
			},
		},

		{
			name: "embedded",
			src:  "package p\ntype T struct{ Foo }",
			want: []Field{{TypeName: "Foo"}},
		},
		{
			name: "embedded_pointer",
			src:  "package p\ntype T struct{ *Foo }",
			want: []Field{{TypeName: "*Foo"}},
		},
		{
			name: "embedded_qualified",
			src:  "package p\ntype T struct{ pkg.Foo }",
			want: []Field{{TypeName: "pkg.Foo"}},
		},
		{
			name: "embedded_with_tag",
			src:  "package p\ntype T struct{ Foo `json:\"foo\"` }",
			want: []Field{{TypeName: "Foo"}},
		},

		{
			name: "slice_type",
			src:  "package p\ntype T struct{ A []int }",
			want: []Field{{Name: "A", TypeName: "[]int"}},
		},
		{
			name: "pointer_type",
			src:  "package p\ntype T struct{ A *Foo }",
			want: []Field{{Name: "A", TypeName: "*Foo"}},
		},
		{
			name: "map_type",
			src:  "package p\ntype T struct{ A map[string]int }",
			want: []Field{{Name: "A", TypeName: "map[string]int"}},
		},
		{
			name: "selector_type",
			src:  "package p\ntype T struct{ A time.Time }",
			want: []Field{{Name: "A", TypeName: "time.Time"}},
		},
		{
			name: "array_type",
			src:  "package p\ntype T struct{ A [10]int }",
			want: []Field{{Name: "A", TypeName: "[10]int"}},
		},
		{
			name: "chan_type",
			src:  "package p\ntype T struct{ C chan int }",
			want: []Field{{Name: "C", TypeName: "chan int"}},
		},
		{
			name: "generic_type",
			src:  "package p\ntype T struct{ G List[int] }",
			want: []Field{{Name: "G", TypeName: "List[int]"}},
		},
		{
			name: "empty_interface_type",
			src:  "package p\ntype T struct{ I interface{} }",
			want: []Field{{Name: "I", TypeName: "interface{}"}},
		},
		{
			name: "func_type",
			src:  "package p\ntype T struct{ F func() error }",
			want: []Field{{Name: "F", TypeName: "func() error"}},
		},

		{
			name: "mixed",
			src:  "package p\ntype T struct{ A int; Foo; B, C string `json:\"x\"` }",
			want: []Field{
				{Name: "A", TypeName: "int"},
				{TypeName: "Foo"},
				{Name: "B", TypeName: "string"},
				{Name: "C", TypeName: "string"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Go 1.22+: tt per-iteration

			st := parseStruct(t, tt.src)
			got := extractFields(st)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractFields:\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestExtractFields_NilFieldList(t *testing.T) {
	t.Parallel()
	st := &ast.StructType{Fields: nil}
	if got := extractFields(st); got != nil {
		t.Errorf("want nil for st.Fields == nil, got %#v", got)
	}
}

func parseStruct(t *testing.T, src string) *ast.StructType {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v\nsrc:\n%s", err, src)
	}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				return st
			}
		}
	}
	t.Fatalf("no struct type in src:\n%s", src)
	return nil
}

package reset

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

// ResetTarget represents a discovered Go struct that requires reset logic generation.
type ResetTarget struct {
	PkgName  string  // Package name
	Line     int     // Line number for precise error messages
	TypeName string  // TypeName is the identifier of the struct
	Fields   []Field // Fields is the  of struct fields in the order they are declared in the source code.
}

// Field represents a single field within a ResetTarget struct.
type Field struct {
	Name     string // Name is the field name
	TypeName string // TypeName representation of the field's type  (e.g., "int", "map[string]string").
}

// FindResetTargets analyzes the specified Go source file and extracts all
// struct declarations preceded by a comment containing the "generate:reset" directive.
func FindResetTargets(filePath string) ([]ResetTarget, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	var targets []ResetTarget
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		if !hasDirective(gen.Doc, "generate:reset") {
			continue
		}

		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			targets = append(targets, ResetTarget{
				PkgName:  f.Name.Name,
				Line:     fset.Position(ts.Pos()).Line,
				TypeName: ts.Name.Name,
				Fields:   extractFields(st),
			})
		}
	}
	return targets, nil
}

func hasDirective(cg *ast.CommentGroup, want string) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		if strings.TrimSpace(c.Text) == "// "+want {
			return true
		}
	}
	return false
}

func extractFields(st *ast.StructType) []Field {
	if st.Fields == nil || st.Fields.NumFields() == 0 {
		return nil
	}
	fields := make([]Field, 0, st.Fields.NumFields())
	for _, f := range st.Fields.List {
		typeName := types.ExprString(f.Type)

		if len(f.Names) == 0 {
			fields = append(fields, Field{TypeName: typeName})
			continue
		}
		for _, name := range f.Names {
			fields = append(fields, Field{Name: name.Name, TypeName: typeName})
		}
	}
	return fields
}

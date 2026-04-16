package expr

import (
	"fmt"
	"strings"
)

// DeclKind distinguishes enum vs range declarations.
type DeclKind int

const (
	DeclEnum  DeclKind = iota
	DeclRange
)

// ExpandDecl is a := declaration that specifies a variable's possible values.
type ExpandDecl struct {
	VarName string
	Kind    DeclKind
	Values  []string // for DeclEnum
	Start   int64    // for DeclRange
	End     int64    // for DeclRange
}

// Count returns the number of possible values.
func (d *ExpandDecl) Count() int {
	switch d.Kind {
	case DeclEnum:
		return len(d.Values)
	case DeclRange:
		return int(d.End - d.Start)
	}
	return 0
}

// ExpandCompute is a = computation that defines a runtime calculation.
type ExpandCompute struct {
	VarName string
	Expr    Expr
	deps    []string // variable dependencies (for topo sort)
}

// ExpandSet holds both declarations and computations parsed from expand_exprs.
type ExpandSet struct {
	Decls    []*ExpandDecl
	Computes []*ExpandCompute
}

// ParseExpands parses a list of expand_exprs strings into an ExpandSet.
// Declarations (:=) and computations (=) are separated and validated.
// Computations are topologically sorted by their variable dependencies.
func ParseExpands(raw []string) (*ExpandSet, error) {
	set := &ExpandSet{}

	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse: ${varName} := ... or ${varName} = ...
		if !strings.HasPrefix(line, "${") {
			return nil, fmt.Errorf("expand expression must start with ${var}: %q", line)
		}

		// Find the closing }
		closeIdx := strings.Index(line, "}")
		if closeIdx < 0 {
			return nil, fmt.Errorf("unclosed ${...} in expand expression: %q", line)
		}
		varName := strings.TrimSpace(line[2:closeIdx])
		if varName == "" {
			return nil, fmt.Errorf("empty variable name in expand expression: %q", line)
		}
		rest := strings.TrimSpace(line[closeIdx+1:])

		if strings.HasPrefix(rest, ":=") {
			// Declaration
			rhs := strings.TrimSpace(rest[2:])
			decl, err := parseDeclRHS(varName, rhs)
			if err != nil {
				return nil, fmt.Errorf("in %q: %w", line, err)
			}
			set.Decls = append(set.Decls, decl)
		} else if strings.HasPrefix(rest, "=") {
			// Computation
			rhs := strings.TrimSpace(rest[1:])
			expr, err := ParseExpressionString(rhs)
			if err != nil {
				return nil, fmt.Errorf("in %q: %w", line, err)
			}
			deps := collectVarRefsFromExpr(expr)
			set.Computes = append(set.Computes, &ExpandCompute{
				VarName: varName,
				Expr:    expr,
				deps:    deps,
			})
		} else {
			return nil, fmt.Errorf("expected '=' or ':=' after ${%s}: %q", varName, line)
		}
	}

	// Validate: check for duplicate := declarations
	declSeen := make(map[string]bool)
	for _, d := range set.Decls {
		if declSeen[d.VarName] {
			return nil, fmt.Errorf("duplicate := declaration for variable ${%s}", d.VarName)
		}
		declSeen[d.VarName] = true
	}

	// Validate: check that all function calls reference registered functions
	for _, c := range set.Computes {
		if err := validateFuncRefs(c.Expr); err != nil {
			return nil, fmt.Errorf("in ${%s} = ...: %w", c.VarName, err)
		}
	}

	// Topological sort the computations
	sorted, err := topoSortComputes(set.Computes)
	if err != nil {
		return nil, err
	}
	set.Computes = sorted

	return set, nil
}

// validateFuncRefs checks that all function calls in an expression reference registered functions.
func validateFuncRefs(e Expr) error {
	switch n := e.(type) {
	case *FuncCall:
		if _, ok := LookupFunc(n.Name); !ok {
			return fmt.Errorf("unknown function %s()", n.Name)
		}
		for _, arg := range n.Args {
			if err := validateFuncRefs(arg); err != nil {
				return err
			}
		}
	case *BinaryOp:
		if err := validateFuncRefs(n.Left); err != nil {
			return err
		}
		return validateFuncRefs(n.Right)
	}
	return nil
}

func collectVarRefsFromExpr(e Expr) []string {
	var refs []string
	switch n := e.(type) {
	case *VarRef:
		refs = append(refs, n.Name)
	case *BinaryOp:
		refs = append(refs, collectVarRefsFromExpr(n.Left)...)
		refs = append(refs, collectVarRefsFromExpr(n.Right)...)
	case *FuncCall:
		for _, arg := range n.Args {
			refs = append(refs, collectVarRefsFromExpr(arg)...)
		}
	}
	return refs
}

// collectColRefsFromExpr recursively collects all @{column} references from an expression.
func collectColRefsFromExpr(e Expr) []string {
	switch n := e.(type) {
	case *ColRef:
		return []string{n.Name}
	case *BinaryOp:
		return append(collectColRefsFromExpr(n.Left), collectColRefsFromExpr(n.Right)...)
	case *FuncCall:
		var refs []string
		for _, arg := range n.Args {
			refs = append(refs, collectColRefsFromExpr(arg)...)
		}
		return refs
	}
	return nil
}

// RequiredColumns returns the deduplicated list of @{column} references
// used across all compute expressions in this ExpandSet.
func (s *ExpandSet) RequiredColumns() []string {
	seen := make(map[string]bool)
	var cols []string
	for _, comp := range s.Computes {
		for _, name := range collectColRefsFromExpr(comp.Expr) {
			if !seen[name] {
				seen[name] = true
				cols = append(cols, name)
			}
		}
	}
	return cols
}

// parseDeclRHS parses the RHS of a := declaration.
// Supports: enum(val1, val2, ...) and range(start, end)
func parseDeclRHS(varName, rhs string) (*ExpandDecl, error) {
	if strings.HasPrefix(rhs, "enum(") && strings.HasSuffix(rhs, ")") {
		inner := rhs[5 : len(rhs)-1]
		parts := strings.Split(inner, ",")
		var values []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			values = append(values, p)
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("enum() requires at least one value")
		}
		return &ExpandDecl{
			VarName: varName,
			Kind:    DeclEnum,
			Values:  values,
		}, nil
	}

	if strings.HasPrefix(rhs, "range(") && strings.HasSuffix(rhs, ")") {
		inner := rhs[6 : len(rhs)-1]
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("range() requires exactly 2 arguments: range(start, end)")
		}
		start, err := parseInt64(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("range() start: %w", err)
		}
		end, err := parseInt64(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("range() end: %w", err)
		}
		if start >= end {
			return nil, fmt.Errorf("range() requires start < end, got range(%d, %d)", start, end)
		}
		return &ExpandDecl{
			VarName: varName,
			Kind:    DeclRange,
			Start:   start,
			End:     end,
		}, nil
	}

	return nil, fmt.Errorf("expected enum(...) or range(...), got %q", rhs)
}

func parseInt64(s string) (int64, error) {
	return intFromString(s)
}

// topoSortComputes performs a topological sort on computations based on variable dependencies.
func topoSortComputes(computes []*ExpandCompute) ([]*ExpandCompute, error) {
	if len(computes) <= 1 {
		return computes, nil
	}

	// Kahn's algorithm
	type node struct {
		compute *ExpandCompute
		index   int
	}

	nodes := make([]node, len(computes))
	nameToIdx := make(map[string]int)
	for i, c := range computes {
		nodes[i] = node{compute: c, index: i}
		nameToIdx[c.VarName] = i
	}

	// Build adjacency: if compute[i] depends on compute[j].VarName, j -> i
	inDegree := make([]int, len(computes))
	adj := make([][]int, len(computes))
	for i, c := range computes {
		for _, dep := range c.deps {
			if j, ok := nameToIdx[dep]; ok && j != i {
				adj[j] = append(adj[j], i)
				inDegree[i]++
			}
		}
	}

	var queue []int
	for i, d := range inDegree {
		if d == 0 {
			queue = append(queue, i)
		}
	}

	var sorted []*ExpandCompute
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		sorted = append(sorted, computes[idx])
		for _, next := range adj[idx] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(sorted) != len(computes) {
		return nil, fmt.Errorf("circular dependency detected in expand expressions")
	}

	return sorted, nil
}

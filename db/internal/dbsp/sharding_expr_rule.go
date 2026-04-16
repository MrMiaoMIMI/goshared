package dbsp

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ================== Expression DB Rule ==================

var (
	_ dbspi.DbShardingRule            = (*exprDbRule)(nil)
	_ dbspi.ShardingKeyColumnsProvider = (*exprDbRule)(nil)
)

type exprDbRule struct {
	tmpl    *expr.Template
	expands *expr.ExpandSet
}

// NewExprDbRule creates a DB sharding rule from a name expression template and expand set.
func NewExprDbRule(tmpl *expr.Template, expands *expr.ExpandSet) *exprDbRule {
	return &exprDbRule{tmpl: tmpl, expands: expands}
}

func (r *exprDbRule) ResolveDbKey(sk *dbspi.ShardingKey) (string, error) {
	if sk == nil {
		return "", dbspi.ErrShardingKeyRequired
	}
	ctx, err := r.buildContext(sk)
	if err != nil {
		return "", err
	}
	return r.tmpl.Eval(ctx)
}

func (r *exprDbRule) RequiredColumns() []string {
	return r.expands.RequiredColumns()
}

func (r *exprDbRule) buildContext(sk *dbspi.ShardingKey) (*expr.EvalContext, error) {
	ctx := expr.NewContext()
	if err := ctx.LoadColumnsFromMap(sk.Fields()); err != nil {
		return nil, fmt.Errorf("load sharding key: %w", err)
	}
	for _, comp := range r.expands.Computes {
		val, err := expr.Eval(comp.Expr, ctx)
		if err != nil {
			return nil, fmt.Errorf("compute ${%s}: %w", comp.VarName, err)
		}
		ctx.SetVar(comp.VarName, val)
	}
	return ctx, nil
}

// EnumerateDbNames generates all possible db names from := declarations.
// Used at startup to create DbTarget connections.
func (r *exprDbRule) EnumerateDbNames() ([]string, error) {
	if len(r.expands.Decls) == 0 {
		return nil, fmt.Errorf("no := declarations found for DB enumeration")
	}

	// Collect all values to enumerate
	return r.enumerateNames(r.expands.Decls)
}

func (r *exprDbRule) enumerateNames(decls []*expr.ExpandDecl) ([]string, error) {
	ctx := expr.NewContext()
	return r.enumerateNamesRecursive(ctx, decls)
}

func (r *exprDbRule) enumerateNamesRecursive(ctx *expr.EvalContext, decls []*expr.ExpandDecl) ([]string, error) {
	if len(decls) == 0 {
		// All declarations bound; evaluate format computes then render template
		for _, comp := range r.expands.Computes {
			val, err := expr.Eval(comp.Expr, ctx)
			if err != nil {
				continue // skip computes that need runtime columns
			}
			ctx.SetVar(comp.VarName, val)
		}
		name, err := r.tmpl.Eval(ctx)
		if err != nil {
			return nil, err
		}
		return []string{name}, nil
	}

	first := decls[0]
	rest := decls[1:]

	var values []expr.Value
	switch first.Kind {
	case expr.DeclEnum:
		for _, v := range first.Values {
			values = append(values, expr.StrValue(v))
		}
	case expr.DeclRange:
		for i := first.Start; i < first.End; i++ {
			values = append(values, expr.IntValue(i))
		}
	}

	var allNames []string
	for _, val := range values {
		childCtx := expr.NewContext()
		childCtx.SetVar(first.VarName, val)
		childCtx.SetCol(first.VarName, val)
		// Copy bindings from parent context
		copyContextVars(ctx, childCtx)
		subNames, err := r.enumerateNamesRecursive(childCtx, rest)
		if err != nil {
			return nil, err
		}
		allNames = append(allNames, subNames...)
	}
	return allNames, nil
}

func copyContextVars(src, dst *expr.EvalContext) {
	src.CopyTo(dst)
}

// ================== Expression Table Rule ==================

var (
	_ dbspi.TableShardingRule          = (*exprTableRule)(nil)
	_ dbspi.ShardCounter               = (*exprTableRule)(nil)
	_ dbspi.ShardEnumerator            = (*exprTableRule)(nil)
	_ dbspi.ShardingKeyColumnsProvider = (*exprTableRule)(nil)
)

type exprTableRule struct {
	tmpl     *expr.Template
	expands  *expr.ExpandSet
	indexVar string
	count    int
}

// NewExprTableRule creates a table sharding rule from expressions.
// The indexVar and count are auto-detected from the := range() declaration.
func NewExprTableRule(tmpl *expr.Template, expands *expr.ExpandSet) (*exprTableRule, error) {
	rule := &exprTableRule{
		tmpl:    tmpl,
		expands: expands,
	}

	// Auto-detect indexVar and count from range declarations
	for _, decl := range expands.Decls {
		if decl.Kind == expr.DeclRange {
			rule.indexVar = decl.VarName
			rule.count = int(decl.End - decl.Start)
			break
		}
	}

	// If no range declaration, try enum declaration
	if rule.indexVar == "" {
		for _, decl := range expands.Decls {
			if decl.Kind == expr.DeclEnum {
				rule.indexVar = decl.VarName
				rule.count = len(decl.Values)
				break
			}
		}
	}

	return rule, nil
}

func (r *exprTableRule) RequiredColumns() []string {
	return r.expands.RequiredColumns()
}

func (r *exprTableRule) ResolveTable(logicalTable string, sk *dbspi.ShardingKey) (string, error) {
	if sk == nil {
		return "", dbspi.ErrShardingKeyRequired
	}
	ctx := expr.NewContext()
	if err := ctx.LoadColumnsFromMap(sk.Fields()); err != nil {
		return "", fmt.Errorf("load sharding key: %w", err)
	}
	for _, comp := range r.expands.Computes {
		val, err := expr.Eval(comp.Expr, ctx)
		if err != nil {
			return "", fmt.Errorf("compute ${%s}: %w", comp.VarName, err)
		}
		ctx.SetVar(comp.VarName, val)
	}
	return r.tmpl.Eval(ctx)
}

func (r *exprTableRule) ShardCount() int {
	return r.count
}

func (r *exprTableRule) ShardName(logicalTable string, index int) (string, error) {
	if index < 0 || index >= r.count {
		return "", fmt.Errorf("shard index %d out of range [0, %d)", index, r.count)
	}

	ctx := expr.NewContext()

	for _, decl := range r.expands.Decls {
		if decl.VarName != r.indexVar {
			continue
		}
		switch decl.Kind {
		case expr.DeclEnum:
			ctx.SetVar(r.indexVar, expr.StrValue(decl.Values[index]))
			ctx.SetCol(r.indexVar, expr.StrValue(decl.Values[index]))
		case expr.DeclRange:
			actualIndex := decl.Start + int64(index)
			ctx.SetVar(r.indexVar, expr.IntValue(actualIndex))
		}
		break
	}

	// Evaluate computes that are satisfiable without column data.
	// This handles format computes like ${index} = fill(${idx}, 8).
	// Routing computes that need @{col} will be skipped.
	for _, comp := range r.expands.Computes {
		val, err := expr.Eval(comp.Expr, ctx)
		if err != nil {
			continue // skip computes that need runtime columns
		}
		ctx.SetVar(comp.VarName, val)
	}

	return r.tmpl.Eval(ctx)
}

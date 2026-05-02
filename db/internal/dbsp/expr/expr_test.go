package expr

import (
	"testing"
)

// ================== Value Tests ==================

func TestValueInt(t *testing.T) {
	v := IntValue(42)
	if !v.IsInt() {
		t.Fatal("expected int value")
	}
	if v.String() != "42" {
		t.Fatalf("expected '42', got %q", v.String())
	}
	n, err := v.Int64()
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
}

func TestValueString(t *testing.T) {
	v := StrValue("hello")
	if !v.IsString() {
		t.Fatal("expected string value")
	}
	if v.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", v.String())
	}
}

func TestValueStringToInt(t *testing.T) {
	v := StrValue("123")
	n, err := v.Int64()
	if err != nil {
		t.Fatal(err)
	}
	if n != 123 {
		t.Fatalf("expected 123, got %d", n)
	}
}

// ================== Lexer Tests ==================

func TestLexerBasicExpression(t *testing.T) {
	tokens, err := TokenizeExpression("@{shop_id} / 1000 % 1000")
	if err != nil {
		t.Fatal(err)
	}
	expected := []TokenKind{TokenColRef, TokenSlash, TokenInt, TokenPercent, TokenInt, TokenEOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, tk := range expected {
		if tokens[i].Kind != tk {
			t.Fatalf("token[%d]: expected kind %d, got %s", i, tk, tokens[i])
		}
	}
}

func TestLexerVarRef(t *testing.T) {
	tokens, err := TokenizeExpression("${idx} + 1")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Kind != TokenVarRef || tokens[0].Val != "idx" {
		t.Fatalf("expected VarRef(idx), got %s", tokens[0])
	}
}

func TestLexerFuncCall(t *testing.T) {
	tokens, err := TokenizeExpression("hash(@{shop_id}) % 1000")
	if err != nil {
		t.Fatal(err)
	}
	// IDENT(hash) LPAREN COL(shop_id) RPAREN PERCENT INT(1000) EOF
	if tokens[0].Kind != TokenIdent || tokens[0].Val != "hash" {
		t.Fatalf("expected IDENT(hash), got %s", tokens[0])
	}
	if tokens[1].Kind != TokenLParen {
		t.Fatalf("expected LPAREN, got %s", tokens[1])
	}
}

func TestLexerDeclAssign(t *testing.T) {
	tokens, err := TokenizeExpression("${idx} := range(0, 1000)")
	if err != nil {
		t.Fatal(err)
	}
	if tokens[0].Kind != TokenVarRef || tokens[0].Val != "idx" {
		t.Fatalf("expected VarRef(idx), got %s", tokens[0])
	}
	if tokens[1].Kind != TokenDeclAssign {
		t.Fatalf("expected DeclAssign, got %s", tokens[1])
	}
}

// ================== Parser Tests ==================

func TestParseSimpleArithmetic(t *testing.T) {
	e, err := ParseExpressionString("@{shop_id} / 1000 % 1000")
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext()
	ctx.SetCol("shop_id", IntValue(123456))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.MustInt64() != 123 {
		t.Fatalf("expected 123, got %d", val.MustInt64())
	}
}

func TestParseFuncCall(t *testing.T) {
	e, err := ParseExpressionString("hash(@{shop_id}) % 1000")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetCol("shop_id", IntValue(42))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !val.IsInt() {
		t.Fatal("expected int result")
	}
}

func TestParseVarRef(t *testing.T) {
	e, err := ParseExpressionString("${idx2} % 1000")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetVar("idx2", IntValue(12345))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.MustInt64() != 345 {
		t.Fatalf("expected 345, got %d", val.MustInt64())
	}
}

func TestParseParentheses(t *testing.T) {
	e, err := ParseExpressionString("(@{a} + @{b}) * 2")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetCol("a", IntValue(3))
	ctx.SetCol("b", IntValue(4))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.MustInt64() != 14 {
		t.Fatalf("expected 14, got %d", val.MustInt64())
	}
}

func TestParseNestedFuncCall(t *testing.T) {
	e, err := ParseExpressionString("fill(hash(@{id}) % 1000, 8)")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetCol("id", IntValue(42))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !val.IsString() {
		t.Fatal("expected string result from fill()")
	}
	t.Logf("nested func result: %s", val.String())
}

// ================== Template Tests ==================

func TestTemplateSimpleVar(t *testing.T) {
	tmpl, err := ParseTemplate("order_${region}_db")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetVar("region", StrValue("SG"))
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "order_SG_db" {
		t.Fatalf("expected 'order_SG_db', got %q", result)
	}
}

func TestTemplateVarRef(t *testing.T) {
	tmpl, err := ParseTemplate("order_db_${idx}")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetVar("idx", IntValue(3))
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "order_db_3" {
		t.Fatalf("expected 'order_db_3', got %q", result)
	}
}

func TestTemplateMultipleVars(t *testing.T) {
	tmpl, err := ParseTemplate("order_${region}_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetVar("region", StrValue("SG"))
	ctx.SetVar("index", StrValue("00000123"))
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "order_SG_tab_00000123" {
		t.Fatalf("expected 'order_SG_tab_00000123', got %q", result)
	}
}

func TestTemplateTreatsColRefAsLiteral(t *testing.T) {
	_, err := ParseTemplate("order_@{region}_db")
	if err != nil {
		t.Fatal("@{} in template should be treated as literal text, not cause an error")
	}
}

func TestTemplateTreatsFuncCallAsLiteral(t *testing.T) {
	// #{...} is no longer special in templates, just literal text
	tmpl, err := ParseTemplate("order_#{fill(1, 8)}")
	if err != nil {
		t.Fatal("#{} in template should be treated as literal text")
	}
	ctx := NewContext()
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "order_#{fill(1, 8)}" {
		t.Fatalf("expected literal '#{fill(1, 8)}' preserved, got %q", result)
	}
}

func TestTemplateRejectsComplexExpr(t *testing.T) {
	_, err := ParseTemplate("order_db_${@{shop_id} % 4}")
	if err == nil {
		t.Fatal("expected error: template should only accept simple ${var} references")
	}
	t.Logf("error: %v", err)
}

func TestTemplateCollectVarRefs(t *testing.T) {
	tmpl, err := ParseTemplate("order_${region}_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	varRefs := tmpl.CollectVarRefs()
	if len(varRefs) != 2 {
		t.Fatalf("expected 2 var refs, got %v", varRefs)
	}
	if varRefs[0] != "region" || varRefs[1] != "index" {
		t.Fatalf("expected [region, index], got %v", varRefs)
	}
}

// ================== Expand Tests ==================

func TestExpandDeclarationEnum(t *testing.T) {
	set, err := ParseExpands([]string{
		`${region} := enum(SG, TH, ID)`,
		`${region} = @{region}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(set.Decls))
	}
	if set.Decls[0].Kind != DeclEnum {
		t.Fatal("expected DeclEnum")
	}
	if len(set.Decls[0].Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(set.Decls[0].Values))
	}
	if len(set.Computes) != 1 {
		t.Fatalf("expected 1 compute, got %d", len(set.Computes))
	}
}

func TestExpandDeclarationRange(t *testing.T) {
	set, err := ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx} = @{shop_id} % 1000`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Decls) != 1 {
		t.Fatalf("expected 1 decl, got %d", len(set.Decls))
	}
	if set.Decls[0].Kind != DeclRange {
		t.Fatal("expected DeclRange")
	}
	if set.Decls[0].Start != 0 || set.Decls[0].End != 1000 {
		t.Fatalf("expected range(0,1000), got range(%d,%d)", set.Decls[0].Start, set.Decls[0].End)
	}
	if set.Decls[0].Count() != 1000 {
		t.Fatalf("expected count=1000, got %d", set.Decls[0].Count())
	}
}

func TestExpandTopoSort(t *testing.T) {
	set, err := ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx} = ${idx2} % 1000`,
		`${idx2} = @{shop_id} / 1000`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Computes) != 2 {
		t.Fatalf("expected 2 computes, got %d", len(set.Computes))
	}
	if set.Computes[0].VarName != "idx2" {
		t.Fatalf("expected first compute to be idx2, got %s", set.Computes[0].VarName)
	}
	if set.Computes[1].VarName != "idx" {
		t.Fatalf("expected second compute to be idx, got %s", set.Computes[1].VarName)
	}
}

func TestExpandCircularDependency(t *testing.T) {
	_, err := ParseExpands([]string{
		`${a} = ${b} + 1`,
		`${b} = ${a} + 1`,
	})
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
}

// ================== Built-in Function Tests ==================

func TestFillFunction(t *testing.T) {
	e, err := ParseExpressionString("fill(${idx}, 8)")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetVar("idx", IntValue(5))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "00000005" {
		t.Fatalf("expected '00000005', got %q", val.String())
	}
}

func TestHashFunction(t *testing.T) {
	e, err := ParseExpressionString("hash(@{id})")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetCol("id", IntValue(42))
	val1, _ := Eval(e, ctx)

	ctx.SetCol("id", IntValue(42))
	val2, _ := Eval(e, ctx)

	if val1.MustInt64() != val2.MustInt64() {
		t.Fatal("hash should be deterministic")
	}
}

func TestLowerUpperFunction(t *testing.T) {
	lower, _ := ParseExpressionString("lower(@{s})")
	upper, _ := ParseExpressionString("upper(@{s})")
	ctx := NewContext()
	ctx.SetCol("s", StrValue("Hello"))

	lv, _ := Eval(lower, ctx)
	if lv.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", lv.String())
	}

	uv, _ := Eval(upper, ctx)
	if uv.String() != "HELLO" {
		t.Fatalf("expected 'HELLO', got %q", uv.String())
	}
}

func TestConcatFunction(t *testing.T) {
	e, _ := ParseExpressionString("concat(@{a}, @{b})")
	ctx := NewContext()
	ctx.SetCol("a", StrValue("hello"))
	ctx.SetCol("b", StrValue("_world"))
	val, _ := Eval(e, ctx)
	if val.String() != "hello_world" {
		t.Fatalf("expected 'hello_world', got %q", val.String())
	}
}

func TestHashNonNegative(t *testing.T) {
	e, _ := ParseExpressionString("hash(@{id}) % 1000")
	ctx := NewContext()
	for i := int64(0); i < 10000; i++ {
		ctx.SetCol("id", IntValue(i))
		val, err := Eval(e, ctx)
		if err != nil {
			t.Fatalf("i=%d: %v", i, err)
		}
		n := val.MustInt64()
		if n < 0 || n >= 1000 {
			t.Fatalf("i=%d: hash() %% 1000 = %d (expected 0..999)", i, n)
		}
	}
}

func TestBareIdentAsStringInFunc(t *testing.T) {
	e, err := ParseExpressionString("lower(SG)")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "sg" {
		t.Fatalf("expected 'sg', got %q", val.String())
	}
}

func TestBareIdentAsStringStandalone(t *testing.T) {
	e, err := ParseExpressionString("concat(order, _, @{region})")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext()
	ctx.SetCol("region", StrValue("SG"))
	val, err := Eval(e, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if val.String() != "order_SG" {
		t.Fatalf("expected 'order_SG', got %q", val.String())
	}
}

// ================== Validation Error Tests ==================

func TestExpandEmptyVarName(t *testing.T) {
	_, err := ParseExpands([]string{`${} = @{shop_id}`})
	if err == nil {
		t.Fatal("expected error for empty variable name")
	}
	t.Logf("error: %v", err)
}

func TestExpandDuplicateDeclaration(t *testing.T) {
	_, err := ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} := range(0, 20)`,
	})
	if err == nil {
		t.Fatal("expected error for duplicate declaration")
	}
	t.Logf("error: %v", err)
}

func TestExpandUnknownFunction(t *testing.T) {
	_, err := ParseExpands([]string{
		`${idx} = unknown_func(@{shop_id}) % 10`,
	})
	if err == nil {
		t.Fatal("expected error for unknown function")
	}
	t.Logf("error: %v", err)
}

func TestExpandRangeInvalid(t *testing.T) {
	_, err := ParseExpands([]string{`${idx} := range(10, 5)`})
	if err == nil {
		t.Fatal("expected error for range(10, 5)")
	}
}

func TestEvalMissingColumn(t *testing.T) {
	e, _ := ParseExpressionString("@{missing_col}")
	ctx := NewContext()
	_, err := Eval(e, ctx)
	if err == nil {
		t.Fatal("expected error for missing column")
	}
	t.Logf("error: %v", err)
}

func TestEvalMissingVariable(t *testing.T) {
	e, _ := ParseExpressionString("${undefined_var}")
	ctx := NewContext()
	_, err := Eval(e, ctx)
	if err == nil {
		t.Fatal("expected error for missing variable")
	}
	t.Logf("error: %v", err)
}

func TestEvalDivisionByZero(t *testing.T) {
	e, _ := ParseExpressionString("@{x} / 0")
	ctx := NewContext()
	ctx.SetCol("x", IntValue(42))
	_, err := Eval(e, ctx)
	if err == nil {
		t.Fatal("expected error for division by zero")
	}
	t.Logf("error: %v", err)
}

// ================== End-to-End Tests ==================

func TestEndToEndRegionDb(t *testing.T) {
	tmpl, err := ParseTemplate("order_${region}_db")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := ParseExpands([]string{
		`${region} := enum(SG, TH, ID)`,
		`${region} = @{region}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext()
	ctx.SetCol("region", StrValue("SG"))
	for _, comp := range expands.Computes {
		val, err := Eval(comp.Expr, ctx)
		if err != nil {
			t.Fatal(err)
		}
		ctx.SetVar(comp.VarName, val)
	}
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result != "order_SG_db" {
		t.Fatalf("expected 'order_SG_db', got %q", result)
	}
}

func TestEndToEndTableSharding(t *testing.T) {
	tmpl, err := ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx2} = @{shop_id} / 1000`,
		`${idx} = ${idx2} % 1000`,
		`${index} = fill(${idx}, 8)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext()
	ctx.SetCol("shop_id", IntValue(123456789))
	for _, comp := range expands.Computes {
		val, err := Eval(comp.Expr, ctx)
		if err != nil {
			t.Fatal(err)
		}
		ctx.SetVar(comp.VarName, val)
	}
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 123456789 / 1000 = 123456, 123456 % 1000 = 456, fill(456,8) = "00000456"
	if result != "order_tab_00000456" {
		t.Fatalf("expected 'order_tab_00000456', got %q", result)
	}
}

func TestEndToEndHashMod(t *testing.T) {
	tmpl, err := ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx} = hash(@{shop_id}) % 1000`,
		`${index} = fill(${idx}, 8)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := NewContext()
	ctx.SetCol("shop_id", IntValue(42))
	for _, comp := range expands.Computes {
		val, err := Eval(comp.Expr, ctx)
		if err != nil {
			t.Fatal(err)
		}
		ctx.SetVar(comp.VarName, val)
	}
	result, err := tmpl.Eval(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	t.Logf("hash-mod result: %s", result)
}

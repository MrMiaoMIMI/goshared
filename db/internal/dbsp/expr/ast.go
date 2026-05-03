package expr

type Expr interface {
	exprNode()
}

type IntLit struct {
	Value int64
}

type StrLit struct {
	Value string
}

type ColRef struct {
	Name string
}

type VarRef struct {
	Name string
}

type BinaryOp struct {
	Op    TokenKind
	Left  Expr
	Right Expr
}

type FuncCall struct {
	Name string
	Args []Expr
}

func (*IntLit) exprNode()   {}
func (*StrLit) exprNode()   {}
func (*ColRef) exprNode()   {}
func (*VarRef) exprNode()   {}
func (*BinaryOp) exprNode() {}
func (*FuncCall) exprNode() {}

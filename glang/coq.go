package glang

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

func addParens(needs_paren bool, expr string) string {
	if needs_paren {
		return "(" + expr + ")"
	} else {
		return expr
	}
}

// buffer is a simple indenting pretty printer
type buffer struct {
	lines       []string
	indentLevel int
}

func (pp buffer) indentation() string {
	b := make([]byte, pp.indentLevel)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

func (pp *buffer) appendLine(line string) {
	pp.lines = append(pp.lines, line)
}

func (pp *buffer) AddLine(line string) {
	if line == "" {
		pp.appendLine("")
	} else {
		pp.appendLine(pp.indentation() + indent(pp.indentLevel, line))
	}
}

// Add adds formatted to the buffer
func (pp *buffer) Add(format string, args ...interface{}) {
	pp.AddLine(fmt.Sprintf(format, args...))
}

func (pp *buffer) Indent(spaces int) {
	pp.indentLevel += spaces
}

func (pp *buffer) Block(prefix string, format string, args ...interface{}) int {
	pp.AddLine(prefix + indent(len(prefix), fmt.Sprintf(format, args...)))
	pp.Indent(len(prefix))
	return len(prefix)
}

func (pp buffer) Build() string {
	return strings.Join(pp.lines, "\n")
}

func indent(spaces int, s string) string {
	lines := strings.Split(s, "\n")
	indentation := strings.Repeat(" ", spaces)
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		lines[i] = indentation + line
	}
	return strings.Join(lines, "\n")
}

func (pp *buffer) AddComment(c string) {
	if c == "" {
		return
	}
	// these hacks ensure that Go comments don't insert stray Coq comments
	c = strings.ReplaceAll(c, "(*", "( *")
	c = strings.ReplaceAll(c, "*)", "* )")
	indent := pp.Block("(* ", "%s *)", c)
	pp.Indent(-indent)
}

func quote(s string) string {
	return `"` + s + `"`
}

// FIXME: why is this needed?
func binder(s string) string {
	if s == "_" {
		return "<>"
	}
	return quote(s)
}

// FieldDecl is a name:type declaration (for a struct or function binders)
type FieldDecl struct {
	Name string
	Type Type
}

func (d FieldDecl) CoqBinder() string {
	return binder(d.Name)
}

func (d FieldDecl) Coq(needs_paren bool) string {
	return binder(d.Name)
}

// StructDecl is a Coq record for a Go struct
type StructType struct {
	Fields []FieldDecl
}

// CoqDecl implements the Decl interface
//
// A struct declaration simply consists of the struct descriptor
func (d StructType) Coq(needs_paren bool) string {
	var pp buffer
	pp.Add("structT [")
	pp.Indent(2)
	for i, fd := range d.Fields {
		sep := ";"
		if i == len(d.Fields)-1 {
			sep = ""
		}
		pp.Add("%s :: %s%s", quote(fd.Name), fd.Type.Coq(false), sep)
	}
	pp.Indent(-2)
	pp.AddLine("]")
	return pp.Build()
}

type TypeDecl struct {
	Name string
	Body Type
}

func (d TypeDecl) CoqDecl() string {
	var pp buffer
	pp.Add("Definition %s : go_type := %s.", d.Name, d.Body.Coq(false))
	return pp.Build()
}

// Type represents some Coq type.
//
// Structurally identical to Expr but serves as a nice annotation in the type
// system for where types are expected.
type Type interface {
	// If needs_paren is true, this should be generated with parentheses.
	Coq(needs_paren bool) string
}

// TypeIdent is an identifier referencing a type.
//
// Much like the Type interface this is the same as Ident but signals that a Go
// type rather than a value is being referenced.
type TypeIdent string

func (t TypeIdent) Coq(needs_paren bool) string {
	return string(t)
}

// StructName refers to a struct type from its name.
//
// This is Type rather than an Expr.
type StructName string

func (t StructName) Coq(needs_paren bool) string {
	return string(t)
}

type MapType struct {
	Key   Type
	Value Type
}

func (t MapType) Coq(needs_paren bool) string {
	return NewCallExpr(GallinaIdent("mapT"), t.Key, t.Value).Coq(needs_paren)
}

type FuncType struct {
}

func (t FuncType) Coq(needs_paren bool) string {
	return "funcT"
}

type SliceType struct {
	Value Type
}

func (t SliceType) Coq(needs_paren bool) string {
	return NewCallExpr(GallinaIdent("sliceT"), t.Value).Coq(needs_paren)
}

type ArrayType struct {
	Len uint64
	Elt Type
}

func (t ArrayType) Coq(needs_paren bool) string {
	return NewCallExpr(GallinaIdent("arrayT"), t.Elt).Coq(needs_paren)
}

type Expr interface {
	// If needs_paren is true, this should be generated with parentheses.
	Coq(needs_paren bool) string
}

// GallinaIdent is a identifier in Gallina (and not a variable)
//
// A GallinaIdent is translated literally to Coq.
type GallinaIdent string

func (e GallinaIdent) Coq(needs_paren bool) string {
	return string(e)
}

// A Go qualified identifier, which is translated to a Gallina qualified
// identifier.
type PackageIdent struct {
	Package string
	Ident   string
}

func (e PackageIdent) Coq(needs_paren bool) string {
	return fmt.Sprintf("%s.%s", thisIsBadAndShouldBeDeprecatedGoPathToCoqPath(e.Package), e.Ident)
}

var Skip Expr = GallinaIdent("Skip")

type LoggingStmt struct {
	GoCall string
}

func (s LoggingStmt) Coq(needs_paren bool) string {
	var pp buffer
	pp.AddComment(s.GoCall)
	return pp.Build()
}

type ParenExpr struct {
	Inner Expr
}

func (e ParenExpr) Coq(needs_paren bool) string {
	return fmt.Sprintf("(%s)", e.Inner.Coq(needs_paren))
}

// IdentExpr is a go_lang-level variable
//
// An IdentExpr is quoted in Coq.
type IdentExpr string

func (e IdentExpr) Coq(needs_paren bool) string {
	return quote(string(e))
}

// GallinaString is a Gallina string, wrapped in quotes
//
// This is functionally identical to IdentExpr, but semantically quite
// different.
type GallinaString string

func (s GallinaString) Coq(needs_paren bool) string {
	return quote(string(s))
}

// CallExpr includes primitives and references to other functions.
type CallExpr struct {
	MethodName Expr
	Args       []Expr
}

// NewCallExpr is a convenience to construct a CallExpr statically, especially
// for a fixed number of arguments.
func NewCallExpr(name Expr, args ...Expr) CallExpr {
	if len(args) == 0 {
		args = []Expr{Tt}
	}
	return CallExpr{MethodName: name, Args: args}
}

func (s CallExpr) Coq(needs_paren bool) string {
	comps := []string{s.MethodName.Coq(true)}

	for _, a := range s.Args {
		comps = append(comps, a.Coq(true))
	}
	return addParens(needs_paren, strings.Join(comps, " "))
}

func StructDesc(name string) Expr {
	return GallinaIdent(name)
}

type ContinueExpr struct {
}

func (e ContinueExpr) Coq(needs_paren bool) string {
	return fmt.Sprintf("continue: #()")
}

type BreakExpr struct {
}

func (e BreakExpr) Coq(needs_paren bool) string {
	return fmt.Sprintf("break: #()")
}

type ReturnExpr struct {
	Value Expr
}

func (e ReturnExpr) Coq(needs_paren bool) string {
	return fmt.Sprintf("return: %s", e.Value.Coq(needs_paren))
}

type DoExpr struct {
	Expr Expr
}

func (b DoExpr) Coq(needs_paren bool) string {
	var pp buffer
	pp.Add("do:  %s", b.Expr.Coq(false))
	return addParens(needs_paren, pp.Build())
}

type LetExpr struct {
	// Names is a list to support anonymous and tuple-destructuring bindings.
	//
	// If Names is an empty list the binding is anonymous.
	Names   []string
	ValExpr Expr
	Cont    Expr
}

func NewDoSeq(e, cont Expr) LetExpr {
	return LetExpr{ValExpr: DoExpr{Expr: e}, Cont: cont}
}

func (e LetExpr) isAnonymous() bool {
	return len(e.Names) == 0
}

func (b LetExpr) Coq(needs_paren bool) string {
	var pp buffer
	// Printing for anonymous and multiple return values
	// TODO: exception monad sequencing should be its own thing, not an anonymous let binding.
	if b.isAnonymous() {
		pp.Add("%s;;;", b.ValExpr.Coq(false))
	} else if len(b.Names) == 1 {
		pp.Add("let: %s := %s in", binder(b.Names[0]), b.ValExpr.Coq(false))
	} else if len(b.Names) == 2 {
		pp.Add("let: (%s, %s) := %s in",
			binder(b.Names[0]),
			binder(b.Names[1]),
			b.ValExpr.Coq(false))
	} else if len(b.Names) == 3 {
		pp.Add("let: ((%s, %s), %s) := %s in",
			binder(b.Names[0]),
			binder(b.Names[1]),
			binder(b.Names[2]),
			b.ValExpr.Coq(false))
	} else if len(b.Names) == 4 {
		pp.Add("let: (((%s, %s), %s), %s) := %s in",
			binder(b.Names[0]),
			binder(b.Names[1]),
			binder(b.Names[2]),
			binder(b.Names[3]),
			b.ValExpr.Coq(false))
	} else {
		panic("no support for destructuring more than 4 return values")
	}
	pp.Add(b.Cont.Coq(false))
	return addParens(needs_paren, pp.Build())
}

type fieldVal struct {
	Field string
	Value Expr
}

// A StructLiteral represents a record literal construction using name fields.
//
// Relies on Coq record syntax to correctly order fields for the record's
// constructor.
type StructLiteral struct {
	StructName string
	elts       []fieldVal
}

// NewStructLiteral creates a StructLiteral with no values.
func NewStructLiteral(structName string) StructLiteral {
	return StructLiteral{StructName: structName}
}

// AddField appends a new (field, val) pair to a StructLiteral.
func (sl *StructLiteral) AddField(field string, value Expr) {
	sl.elts = append(sl.elts, fieldVal{field, value})
}

func (sl StructLiteral) Coq(needs_paren bool) string {
	var pp buffer
	method := "struct.make"
	pp.Add("%s %s [{", method, StructDesc(sl.StructName).Coq(true))
	pp.Indent(2)
	for i, f := range sl.elts {
		terminator := ";"
		if i == len(sl.elts)-1 {
			terminator = ""
		}
		pp.Add("%s ::= %s%s", quote(f.Field), f.Value.Coq(false), terminator)
	}
	pp.Indent(-2)
	pp.Add("}]")
	return addParens(needs_paren, pp.Build())
}

type BoolLiteral bool

var (
	False BoolLiteral = false
	True  BoolLiteral = true
)

func (b BoolLiteral) Coq(needs_paren bool) string {
	if b {
		return "#true"
	} else {
		return "#false"
	}
}

type UnitLiteral struct{}

var Tt UnitLiteral = struct{}{}

func (tt UnitLiteral) Coq(needs_paren bool) string {
	return "#()"
}

type IntLiteral struct {
	Value uint64
}

func (l IntLiteral) Coq(needs_paren bool) string {
	return fmt.Sprintf("#%d", l.Value)
}

type Int32Literal struct {
	Value uint32
}

func (l Int32Literal) Coq(needs_paren bool) string {
	return fmt.Sprintf("#(U32 %d)", l.Value)
}

type ByteLiteral struct {
	Value uint8
}

func (l ByteLiteral) Coq(needs_paren bool) string {
	return fmt.Sprintf("#(U8 %v)", l.Value)
}

type StringLiteral struct {
	Value string
}

func (l StringLiteral) Coq(needs_paren bool) string {
	return fmt.Sprintf(`#(str "%s")`, l.Value)
}

type nullLiteral struct{}

func (l nullLiteral) Coq(needs_paren bool) string {
	return "#null"
}

// Null represents a nil pointer in Go
var Null = nullLiteral{}

// BinOp is an enum for a Coq binary operator
type BinOp int

// Constants for the supported Coq binary operators
const (
	OpPlus BinOp = iota
	OpMinus
	OpEquals
	OpNotEquals
	OpLessThan
	OpGreaterThan
	OpLessEq
	OpGreaterEq
	OpAppend
	OpMul
	OpQuot
	OpRem
	OpAnd
	OpOr
	OpXor
	OpLAnd
	OpLOr
	OpShl
	OpShr
)

type BinaryExpr struct {
	X  Expr
	Op BinOp
	Y  Expr
}

func (be BinaryExpr) Coq(needs_paren bool) string {
	coqBinOp := map[BinOp]string{
		OpPlus:        "+",
		OpMinus:       "-",
		OpEquals:      "=",
		OpNotEquals:   "≠",
		OpAppend:      "+",
		OpMul:         "*",
		OpQuot:        "`quot`",
		OpRem:         "`rem`",
		OpLessThan:    "<",
		OpGreaterThan: ">",
		OpLessEq:      "≤",
		OpGreaterEq:   "≥",
		OpAnd:         "`and`",
		OpOr:          "`or`",
		OpXor:         "`xor`",
		OpLAnd:        "&&",
		OpLOr:         "||",
		OpShl:         "≪",
		OpShr:         "≫",
	}
	if binop, ok := coqBinOp[be.Op]; ok {
		expr := fmt.Sprintf("%s %s %s",
			be.X.Coq(true), binop, be.Y.Coq(true))
		return addParens(needs_paren, expr)
	}

	panic(fmt.Sprintf("unknown binop %d", be.Op))
}

type NotExpr struct {
	X Expr
}

func (e NotExpr) Coq(needs_paren bool) string {
	return fmt.Sprintf("(~ %s)", e.X.Coq(true))
}

type TupleExpr []Expr

func (te TupleExpr) Coq(needs_paren bool) string {
	var comps []string
	for _, t := range te {
		comps = append(comps, t.Coq(false))
	}
	return fmt.Sprintf("(%s)",
		indent(1, strings.Join(comps, ", ")))
}

// NewTuple is a smart constructor that wraps multiple expressions in a TupleExpr
func NewTuple(es []Expr) Expr {
	if len(es) == 1 {
		return es[0]
	}
	return TupleExpr(es)
}

type ListExpr []Expr

func (le ListExpr) Coq(needs_paren bool) string {
	var comps []string
	for _, t := range le {
		comps = append(comps, t.Coq(false))
	}
	elements := indent(1, strings.Join(comps, "; "))
	if strings.HasPrefix(elements, "#") {
		// [# ...] is a vector notation while we want the list notation [ ... ]
		// (and `[#` is a single token even if the vector notation isn't in
		// scope)
		return fmt.Sprintf("[ %s ]", elements)
	}
	return fmt.Sprintf("[%s]", elements)
}

type DerefExpr struct {
	X  Expr
	Ty Expr
}

func (e DerefExpr) Coq(needs_paren bool) string {
	expr := fmt.Sprintf("![%s] %s", e.Ty.Coq(false), e.X.Coq(true))
	return addParens(needs_paren, expr)
}

type RefExpr struct {
	X  Expr
	Ty Expr
}

func (e RefExpr) Coq(needs_paren bool) string {
	return NewCallExpr(GallinaIdent("ref_ty"), e.Ty, e.X).Coq(needs_paren)
}

type StoreStmt struct {
	Dst Expr
	Ty  Expr
	X   Expr
}

func (e StoreStmt) Coq(needs_paren bool) string {
	expr := fmt.Sprintf("%s <-[%s] %s", e.Dst.Coq(true), e.Ty.Coq(false), e.X.Coq(true))
	return addParens(needs_paren, expr)
}

type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

func flowBranch(pp *buffer, prefix string, e Expr, suffix string) {
	code := e.Coq(false) + suffix
	if !strings.ContainsRune(code, '\n') {
		// compact, single-line form
		indent := pp.Block(prefix+" ", "%s", code)
		pp.Indent(-indent)
		return
	}
	// full multiline, nicely indented form
	pp.AddLine(prefix)
	pp.Indent(2)
	pp.AddLine(code)
	pp.Indent(-2)
}

func (ife IfExpr) Coq(needs_paren bool) string {
	var pp buffer
	// Since we are parenthesesizing all if, we don't need to parenthesize the things inside the if
	pp.Add("(if: %s", ife.Cond.Coq(false))
	flowBranch(&pp, "then", ife.Then, "")
	flowBranch(&pp, "else", ife.Else, ")")
	return pp.Build()
}

// The init statement must wrap the ForLoopExpr, so it can make use of bindings
// introduced there.
type ForLoopExpr struct {
	Cond Expr
	Post Expr
	// the body of the loop
	Body Expr
}

func (e ForLoopExpr) Coq(needs_paren bool) string {
	var pp buffer
	pp.Add("(for: (λ: <>, %s); (λ: <>, %s) := λ: <>,", e.Cond.Coq(false), e.Post.Coq(false))
	pp.Indent(2)
	pp.Add("%s)", e.Body.Coq(false))
	return pp.Build()
}

type ForRangeSliceExpr struct {
	Key   Binder
	Val   Binder
	Ty    Expr
	Slice Expr
	Body  Expr
}

func (e ForRangeSliceExpr) Coq(needs_paren bool) string {
	var pp buffer
	pp.Add("slice.for_range %s %s (λ: %s %s,",
		e.Ty.Coq(true),
		e.Slice.Coq(true),
		binderToCoq(e.Key), binderToCoq(e.Val),
	)
	pp.Indent(2)
	if e.Key != nil && *e.Key != "_" {
		pp.Add("let: %s := ref_ty uint64T %s in", binderToCoq(e.Key), binderToCoq(e.Key))
	}
	if e.Val != nil && *e.Val != "_" {
		pp.Add("let: %s := ref_ty %s %s in", binderToCoq(e.Val), e.Ty.Coq(true), binderToCoq(e.Val))
	}
	pp.Add("%s)", e.Body.Coq(false))
	pp.Indent(-2)
	return addParens(needs_paren, pp.Build())
}

type Binder *IdentExpr

func binderToCoq(b Binder) string {
	if b == nil {
		return "<>"
	}
	return binder(string(*b))
}

// ForRangeMapExpr is a call to the map iteration helper.
type ForRangeMapExpr struct {
	// name of key and value identifiers
	KeyIdent, ValueIdent string
	// map to iterate over
	Map Expr
	// body of loop, with KeyIdent and ValueIdent as free variables
	Body Expr
}

func (e ForRangeMapExpr) Coq(needs_paren bool) string {
	var pp buffer
	pp.Add("MapIter %s (λ: %s %s,",
		e.Map.Coq(true),
		binder(e.KeyIdent), binder(e.ValueIdent))
	pp.Indent(2)
	pp.Add("%s)", e.Body.Coq(false))
	return addParens(needs_paren, pp.Build())
}

// SpawnExpr is a call to Spawn a thread running a procedure.
//
// The body can capture variables in the environment.
type SpawnExpr struct {
	Body Expr
}

func (e SpawnExpr) Coq(needs_paren bool) string {
	var pp buffer
	pp.Block("Fork (", "%s)", e.Body.Coq(false))
	return addParens(needs_paren, pp.Build())
}

// FuncLit is an unnamed function literal, consisting of its parameters and body.
type FuncLit struct {
	Args []FieldDecl
	// TODO: ReturnType Type
	Body Expr
	// TODO: AddTypes   bool
}

func (e FuncLit) Coq(needs_paren bool) string {
	var pp buffer

	var args []string
	for _, a := range e.Args {
		args = append(args, a.CoqBinder())
	}
	if len(args) == 0 {
		args = []string{"<>"}
	}
	sig := strings.Join(args, " ")

	pp.Add("(λ: %s,", sig)
	pp.Indent(2)
	defer pp.Indent(-2)
	pp.AddLine(e.Body.Coq(false))
	pp.Add(")")

	return pp.Build()
}

// FuncDecl declares a function, including its parameters and body.
type FuncDecl struct {
	Name       string
	TypeParams []TypeIdent
	RecvArg    *FieldDecl
	Args       []FieldDecl
	ReturnType Type
	Body       Expr
	Comment    string
	AddTypes   bool
}

// Signature renders the function declaration's bindings
func (d FuncDecl) Signature() string {
	var args []string
	if d.RecvArg != nil {
		args = append(args, d.RecvArg.CoqBinder())
	}
	for _, a := range d.Args {
		args = append(args, a.CoqBinder())
	}
	if len(d.Args) == 0 {
		args = append(args, "<>")
	}
	return strings.Join(args, " ")
}

func (d FuncDecl) Type() string {
	// FIXME: doesn't deal with type parameters
	types := []string{}
	for _, a := range d.Args {
		types = append(types, a.Type.Coq(true))
	}
	if len(d.Args) == 0 {
		types = append(types, TypeIdent("unitT").Coq(true))
	}
	types = append(types, d.ReturnType.Coq(true))
	panic("include RecvArg")
	// return strings.Join(types, " -> ")
}

// CoqDecl implements the Decl interface
//
// For FuncDecl this emits the Coq vernacular Definition that defines the whole
// function.
func (d FuncDecl) CoqDecl() string {
	var pp buffer
	pp.AddComment(d.Comment)

	typeParams := ""
	for _, t := range d.TypeParams {
		typeParams += fmt.Sprintf("(%s: go_type) ", t.Coq(false))
	}

	pp.Add("Definition %s %s: val :=", d.Name, typeParams)
	func() {
		pp.Indent(2)
		defer pp.Indent(-2)
		pp.Add("rec: \"%s\" %s :=", d.Name, d.Signature())
		pp.Indent(2)
		defer pp.Indent(-2)
		pp.AddLine(d.Body.Coq(false) + ".")
	}()
	if d.AddTypes {
		pp.Add("Theorem %s_t: ⊢ %s : (%s).", d.Name, d.Name, d.Type())
		pp.AddLine("Proof. typecheck. Qed.")
		pp.Add("Hint Resolve %s_t : types.", d.Name)
	}
	return pp.Build()
}

// CommentDecl is a top-level comment
//
// Pretends to be a declaration so it can sit among declarations within a file.
type CommentDecl string

// NewComment creates a CommentDecl with proper whitespacing.
func NewComment(s string) CommentDecl {
	comment := strings.TrimRight(s, " \t\n")
	return CommentDecl(comment)
}

// CoqDecl implements the Decl interface
//
// For CommentDecl this emits a Coq top-level comment.
func (d CommentDecl) CoqDecl() string {
	var pp buffer
	pp.AddComment(string(d))
	return pp.Build()
}

type ConstDecl struct {
	Name     string
	Type     Type
	Val      Expr
	Comment  string
	AddTypes bool
}

func (d ConstDecl) CoqDecl() string {
	var pp buffer
	pp.AddComment(d.Comment)
	indent := pp.Block("Definition ", "%s : expr := %s.",
		d.Name, d.Val.Coq(false))
	pp.Indent(-indent)
	if d.AddTypes {
		pp.Add("Theorem %s_t Γ : Γ ⊢ %s : %s.",
			d.Name, d.Name, d.Type.Coq(true))
		pp.AddLine("Proof. typecheck. Qed.")
	}
	return pp.Build()
}

// Decl is a FuncDecl, StructDecl, CommentDecl, or ConstDecl
type Decl interface {
	CoqDecl() string
}

type TupleType []Type

// NewTupleType is a smart constructor that wraps multiple types in a TupleType
func NewTupleType(types []Type) Type {
	if len(types) == 1 {
		return types[0]
	}
	return TupleType(types)
}

func (tt TupleType) Coq(needs_paren bool) string {
	var comps []string
	for _, t := range tt {
		comps = append(comps, t.Coq(true))
	}
	return fmt.Sprintf("(%s)", strings.Join(comps, " * "))
}

type PtrType struct {
}

func (t PtrType) Coq(needs_paren bool) string {
	return "ptrT"
}

func TypeMethod(typeName string, methodName string) string {
	return fmt.Sprintf("%s__%s", typeName, methodName)
}

func InterfaceMethod(interfaceName string, methodName string) string {
	return fmt.Sprintf("(struct.get %s \"%s\")", interfaceName, methodName)
}

const importHeader string = `
From New.golang Require Import defn.
`

// These will not end up in `File.Decls`, they are put into `File.Imports` by `translatePackage`.
type ImportDecl struct {
	Path string
}

// This is an injective mapping
// FIXME: should really use this
func goPathToCoqPath(p string) string {
	p = strings.ReplaceAll(p, "_", "__")
	p = strings.ReplaceAll(p, ".", "_dot_")
	p = strings.ReplaceAll(p, "-", "_dash_")
	return p
}

func thisIsBadAndShouldBeDeprecatedGoPathToCoqPath(p string) string {
	p = strings.ReplaceAll(p, "_", "_")
	p = strings.ReplaceAll(p, ".", "_")
	p = strings.ReplaceAll(p, "-", "_")
	return p
}

// ImportToPath converts a Go import path to a Coq path
//
// TODO: we basically don't handle the package name (determined by the package
//
//	statement in Go) differing from the basename of its parent directory
func ImportToPath(pkgPath, pkgName string) string {
	coqPath := thisIsBadAndShouldBeDeprecatedGoPathToCoqPath(pkgPath)
	p := path.Dir(coqPath)
	filename := path.Base(coqPath) + ".v"
	return filepath.Join(p, filename)
}

func (decl ImportDecl) CoqDecl() string {
	coqImportQualid := strings.ReplaceAll(thisIsBadAndShouldBeDeprecatedGoPathToCoqPath(decl.Path), "/", ".")
	return fmt.Sprintf("From New.code Require %s.", coqImportQualid)
}

// ImportDecls groups imports into one declaration so they can be printed
// without intervening blank spaces.
type ImportDecls []ImportDecl

func (decls ImportDecls) PrintImports() string {
	seen := make(map[string]bool)
	var ss []string
	for _, decl := range decls {
		coqdecl := decl.CoqDecl()
		if !seen[coqdecl] {
			ss = append(ss, coqdecl)
			seen[coqdecl] = true
		}
	}
	sort.Strings(ss)
	return strings.Join(ss, "\n")
}

// File represents a complete Coq file (a sequence of declarations).
type File struct {
	ImportHeader string
	Footer       string
	PkgPath      string
	GoPackage    string
	Imports      ImportDecls
	Decls        []Decl
}

func (f File) autogeneratedNotice() CommentDecl {
	comment := fmt.Sprintf("autogenerated from %s", f.PkgPath)
	return CommentDecl(comment)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Write outputs the Coq source for a File.
// noinspection GoUnhandledErrorResult
func (f File) Write(w io.Writer) {
	fmt.Fprintln(w, f.autogeneratedNotice().CoqDecl())
	fmt.Fprintln(w, strings.Trim(importHeader, "\n"))
	fmt.Fprintln(w, f.Imports.PrintImports())
	if len(f.Imports) > 0 {
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, f.ImportHeader)
	fmt.Fprintln(w)
	decls := make(map[string]bool)
	for i, d := range f.Decls {
		decl := d.CoqDecl()
		// don't translate the same thing twice (which the interface translation
		// can currently do)
		_, isComment := d.(CommentDecl)
		if isComment || !decls[decl] {
			fmt.Fprintln(w, decl)
			decls[decl] = true

			if i != len(f.Decls)-1 {
				fmt.Fprintln(w)
			}
		}
	}
	fmt.Fprint(w, f.Footer)
}

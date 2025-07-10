package main

import (
	"fmt"
	"strconv"
)

type CodeGen struct {
	output     string
	depth      int
	labelCount int
	frameSize  int
}

func newCodeGen() *CodeGen {
	return &CodeGen{
		output:     "",
		depth:      0,
		labelCount: 0,
	}
}

func (cg *CodeGen) println(format string, args ...interface{}) {
	cg.output += fmt.Sprintf(format, args...) + "\n"
}

func (cg *CodeGen) count() int {
	cg.labelCount++
	return cg.labelCount
}

type LocalVars struct {
	variables map[string]int
	stackSize int
}

func newLocalVars() *LocalVars {
	return &LocalVars{
		variables: make(map[string]int),
		stackSize: 0,
	}
}

func (lv *LocalVars) addVariable(name string) int {
	lv.stackSize += 8
	offset := lv.stackSize
	lv.variables[name] = offset
	return offset
}

func (lv *LocalVars) getOffset(name string) (int, bool) {
	offset, exists := lv.variables[name]
	return offset, exists
}

func (cg *CodeGen) push() {
	offset := 128 + 16*cg.depth
	cg.emitStore("x0", "sp", offset)
	cg.depth++
}

func (cg *CodeGen) pop(reg string) {
	cg.depth--
	offset := 128 + 16*cg.depth
	cg.emitLoad(reg, "sp", offset)
}

func alignTo(n, align int) int {
	return (n + align - 1) / align * align
}

func (cg *CodeGen) cmpZero(size int) {
	if size <= 4 {
		cg.println("  cmp w0, #0")
	} else {
		cg.println("  cmp x0, #0")
	}
}

func (cg *CodeGen) pushArgs(args []Expr, params []string, localVars *LocalVars) int {
	stackArgs := 0
	stackArgNum := 0
	if len(args) > 8 {
		stackArgNum = len(args) - 8
		stackArgSize := alignTo(stackArgNum*8, 16)
		cg.println("  sub sp, sp, #%d", stackArgSize)
		for i := 8; i < len(args); i++ {
			cg.genExpr(args[i], params, localVars)
			cg.emitStore("x0", "sp", (i-8)*8)
			stackArgs++
		}
	}

	for i := 0; i < len(args) && i < 8; i++ {
		cg.genExpr(args[i], params, localVars)
		cg.push()
	}

	for i := min(len(args), 8) - 1; i >= 0; i-- {
		reg := fmt.Sprintf("x%d", i)
		cg.pop(reg)
	}

	return stackArgs
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (cg *CodeGen) genExpr(expr Expr, params []string, localVars *LocalVars) {
	switch e := expr.(type) {
	case *ExprIntLiteral:
		cg.println("  mov x0, #%d", e.val)

	case *ExprId:
		paramIndex := getParamIndex(e.name, params)
		if paramIndex >= 0 && paramIndex <= 7 {
			fixed := paramIndex*8 - cg.frameSize
			cg.emitLoad("x0", "x29", fixed)
		} else if paramIndex >= 8 && paramIndex <= 11 {
			offset := 16 + 8*(paramIndex-8)
			cg.emitLoad("x0", "x29", offset)
		} else {
			if offset, exists := localVars.getOffset(e.name); exists {
				actualOffset := len(params)*8 + offset - 8
				cg.emitLoad("x0", "sp", actualOffset)
			}
		}

	case *ExprOp:
		if len(e.args) == 1 {
			cg.genUnaryOp(e.op, e.args[0], params, localVars)
		} else if len(e.args) == 2 {
			cg.genBinaryOp(e.op, e.args[0], e.args[1], params, localVars)
		}

	case *ExprCall:
		cg.genFunctionCall(e, params, localVars)
	}
}

func (cg *CodeGen) genUnaryOp(op string, arg Expr, params []string, localVars *LocalVars) {
	cg.genExpr(arg, params, localVars)

	switch op {
	case "-":
		cg.println("  neg x0, x0")
	case "!":
		cg.cmpZero(8)
		cg.println("  cset x0, eq")
	case "~":
		cg.println("  mvn x0, x0")
	}
}

func (cg *CodeGen) genBinaryOp(op string, left, right Expr, params []string, localVars *LocalVars) {
	switch op {
	case "=":
		cg.genExpr(right, params, localVars)
		if leftId, ok := left.(*ExprId); ok {
			if offset, exists := localVars.getOffset(leftId.name); exists {
				actualOffset := len(params)*8 + offset - 8
				cg.emitStore("x0", "sp", actualOffset)
			}
		}
		return

	default:
		cg.genExpr(left, params, localVars)
		cg.push()
		cg.genExpr(right, params, localVars)
		cg.println("  mov x1, x0")
		cg.pop("x0")

		switch op {
		case "+":
			cg.println("  add x0, x0, x1")
		case "-":
			cg.println("  sub x0, x0, x1")
		case "*":
			cg.println("  mul x0, x0, x1")
		case "/":
			cg.println("  sdiv x0, x0, x1")
		case "%":
			cg.println("  sdiv x2, x0, x1")
			cg.println("  msub x0, x2, x1, x0")
		case "<<":
			cg.println("  lsl x0, x0, x1")
		case ">>":
			cg.println("  asr x0, x0, x1")
		case "&":
			cg.println("  and x0, x0, x1")
		case "|":
			cg.println("  orr x0, x0, x1")
		case "^":
			cg.println("  eor x0, x0, x1")
		case "==":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, eq")
		case "!=":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, ne")
		case "<":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, lt")
		case "<=":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, le")
		case ">":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, gt")
		case ">=":
			cg.println("  cmp x0, x1")
			cg.println("  cset x0, ge")
		case "&&":
			c := cg.count()
			cg.cmpZero(8)
			cg.println("  beq .L.false.%d", c)
			cg.println("  mov x0, x1")
			cg.cmpZero(8)
			cg.println("  beq .L.false.%d", c)
			cg.println("  mov x0, #1")
			cg.println("  b .L.end.%d", c)
			cg.println(".L.false.%d:", c)
			cg.println("  mov x0, #0")
			cg.println(".L.end.%d:", c)
		case "||":
			c := cg.count()
			cg.cmpZero(8)
			cg.println("  bne .L.true.%d", c)
			cg.println("  mov x0, x1")
			cg.cmpZero(8)
			cg.println("  bne .L.true.%d", c)
			cg.println("  mov x0, #0")
			cg.println("  b .L.end.%d", c)
			cg.println(".L.true.%d:", c)
			cg.println("  mov x0, #1")
			cg.println(".L.end.%d:", c)
		}
	}
}

func (cg *CodeGen) genFunctionCall(call *ExprCall, params []string, localVars *LocalVars) {
	_ = cg.pushArgs(call.args, params, localVars)
	stackArgNum := 0
	if len(call.args) > 8 {
		stackArgNum = len(call.args) - 8
	}

	if funId, ok := call.fun.(*ExprId); ok {
		cg.println("  bl %s", funId.name)
	} else {
		cg.genExpr(call.fun, params, localVars)
		cg.println("  blr x0")
	}

	if stackArgNum > 0 {
		cg.println("  add sp, sp, #%d", alignTo(stackArgNum*8, 16))
	}
}

func (cg *CodeGen) genStmt(stmt Stmt, params []string, localVars *LocalVars) {
	switch s := stmt.(type) {
	case *StmtReturn:
		if s.expr != nil {
			cg.genExpr(s.expr, params, localVars)
		}
		totalStackSize := len(params)*8 + localVars.stackSize + 256
		alignedSize := alignTo(totalStackSize, 16)
		cg.println("  add sp, sp, #%d", alignedSize)
		cg.println("  ldp x29, x30, [sp], #16")
		cg.println("  ret")

	case *StmtCompound:
		for _, decl := range s.decls {
			cg.genDecl(decl, localVars)
		}
		for _, stmt := range s.stmts {
			cg.genStmt(stmt, params, localVars)
		}

	case *StmtIf:
		c := cg.count()
		cg.genExpr(s.cond, params, localVars)
		cg.cmpZero(8)
		cg.println("  beq .L.else.%d", c)
		cg.genStmt(s.then_stmt, params, localVars)
		cg.println("  b .L.end.%d", c)
		cg.println(".L.else.%d:", c)
		if s.else_stmt != nil {
			cg.genStmt(s.else_stmt, params, localVars)
		}
		cg.println(".L.end.%d:", c)

	case *StmtWhile:
		c := cg.count()
		cg.println(".L.begin.%d:", c)
		cg.genExpr(s.cond, params, localVars)
		cg.cmpZero(8)
		cg.println("  beq .L.end.%d", c)
		cg.genStmt(s.body, params, localVars)
		cg.println("  b .L.begin.%d", c)
		cg.println(".L.end.%d:", c)

	case *StmtExpr:
		cg.genExpr(s.expr, params, localVars)
	case *StmtFor:
		c := cg.count()
		// 初期化
		cg.genStmt(s.init, params, localVars)

		cg.println(".L.begin.%d:", c)         // ループ条件判定位置
		cg.genExpr(s.cond, params, localVars) // cond
		cg.cmpZero(8)
		cg.println("  beq .L.end.%d", c) // false で脱出

		cg.genStmt(s.body, params, localVars) // body
		cg.genStmt(s.post, params, localVars) // post
		cg.println("  b .L.begin.%d", c)      // 再判定へ
		cg.println(".L.end.%d:", c)

	case *StmtDeclInit:
		cg.genExpr(s.init, params, localVars) // 初期値計算 → x0
		if off, ok := localVars.getOffset(s.decl.name); ok {
			actual := len(params)*8 + off - 8
			cg.emitStore("x0", "sp", actual) // スタックに保存
		}
	}
}

func (cg *CodeGen) genDecl(decl *Decl, localVars *LocalVars) {
}

func (cg *CodeGen) genFunction(fun *DefFun) {
	paramNames := make([]string, len(fun.params))
	for i, decl := range fun.params {
		paramNames[i] = decl.name
	}

	localVars := newLocalVars()
	collectDecls(fun.body, localVars)

	cg.println(".globl %s", fun.name)
	cg.println(".type %s, @function", fun.name)
	cg.println("%s:", fun.name)

	cg.println("  stp x29, x30, [sp, #-16]!")
	cg.println("  mov x29, sp")

	totalStackSize := len(paramNames)*8 + localVars.stackSize + 256
	alignedSize := alignTo(totalStackSize, 16)
	cg.frameSize = alignedSize
	cg.println("  sub sp, sp, #%d", alignedSize)

	for i := range paramNames {
		if i < 8 {
			reg := getParamRegister(i)
			offset := i * 8
			cg.emitStore(reg, "sp", offset)
		}
	}

	cg.genStmt(fun.body, paramNames, localVars)
}

func ast_to_asm_program(program *Program) string {
	if len(program.defs) == 0 {
		return ""
	}

	cg := newCodeGen()
	cg.println(".data")
	cg.println(".text")

	for _, def := range program.defs {
		if d, ok := def.(*DefFun); ok {
			cg.genFunction(d)
		}
	}

	return cg.output
}

func getParamIndex(name string, params []string) int {
	for i, n := range params {
		if n == name {
			return i
		}
	}
	if len(name) >= 2 && name[0] == 'a' {
		if idx, err := strconv.Atoi(name[1:]); err == nil && idx >= 0 && idx <= 11 {
			return idx
		}
	}
	return -1
}

func getParamRegister(index int) string {
	registers := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7"}
	if index >= 0 && index < len(registers) {
		return registers[index]
	}
	return "x0"
}

func (cg *CodeGen) emitLoad(dst, base string, offset int) {
	if offset >= -256 && offset <= 255 {
		cg.println("  ldr %s, [%s, #%d]", dst, base, offset)
	} else {
		if offset < 0 {
			cg.println("  sub x9, %s, #%d", base, -offset)
		} else {
			cg.println("  add x9, %s, #%d", base, offset)
		}
		cg.println("  ldr %s, [x9]", dst)
	}
}

func (cg *CodeGen) emitStore(src, base string, offset int) {
	if offset >= -256 && offset <= 255 {
		cg.println("  str %s, [%s, #%d]", src, base, offset)
	} else {
		if offset < 0 {
			cg.println("  sub x9, %s, #%d", base, -offset)
		} else {
			cg.println("  add x9, %s, #%d", base, offset)
		}
		cg.println("  str %s, [x9]", src)
	}
}

func collectDecls(st Stmt, lv *LocalVars) {
	switch s := st.(type) {
	case *StmtCompound:
		for _, d := range s.decls {
			lv.addVariable(d.name)
		}
		for _, sub := range s.stmts {
			collectDecls(sub, lv)
		}
	case *StmtIf:
		collectDecls(s.then_stmt, lv)
		if s.else_stmt != nil {
			collectDecls(s.else_stmt, lv)
		}
	case *StmtWhile:
		collectDecls(s.body, lv)
	case *StmtFor:
		collectDecls(s.body, lv)
	case *StmtDeclInit:
		lv.addVariable(s.decl.name)
	}
}

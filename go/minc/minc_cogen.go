package main

import (
	"fmt"
	"strconv"
)

// コード生成の状態管理
type CodeGen struct {
	output     string
	depth      int
	labelCount int
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

// ローカル変数のスタックオフセット管理
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

// LocalVarsのaddVariableは0,8,16...と増やす
func (lv *LocalVars) addVariable(name string) int {
	lv.stackSize += 8      // 8バイト（64ビット）確保
	offset := lv.stackSize // 正の値として保存
	lv.variables[name] = offset
	return offset
}

func (lv *LocalVars) getOffset(name string) (int, bool) {
	offset, exists := lv.variables[name]
	return offset, exists
}

// スタック操作
func (cg *CodeGen) push() {
	// 一時的なスタック領域（パラメータ+ローカル変数の後）を使用
	offset := 128 + 16*cg.depth // 128バイト以降を一時領域として使用
	cg.println("  str x0, [sp, #%d]", offset)
	cg.depth++
}

func (cg *CodeGen) pop(reg string) {
	cg.depth--
	offset := 128 + 16*cg.depth
	cg.println("  ldr %s, [sp, #%d]", reg, offset)
}

// アライメント関数
func alignTo(n, align int) int {
	return (n + align - 1) / align * align
}

// レジスタ名の取得
func getRegName(size int, isUnsigned bool) string {
	switch size {
	case 1:
		if isUnsigned {
			return "w0"
		}
		return "w0"
	case 2:
		if isUnsigned {
			return "w0"
		}
		return "w0"
	case 4:
		return "w0"
	case 8:
		return "x0"
	default:
		return "x0"
	}
}

// アドレス生成
func (cg *CodeGen) genAddr(expr Expr, params []string, localVars *LocalVars) {
	switch e := expr.(type) {
	case *ExprId:
		// 変数のアドレス
		paramIndex := getParamIndex(e.name, params)
		if paramIndex >= 0 && paramIndex <= 7 {
			total := len(params)*8 + localVars.stackSize + 256
			aligned := alignTo(total, 16)
			offset := paramIndex*8 - aligned
			if offset < 0 {
				cg.println("  sub x0, x29, #%d", -offset)
			} else {
				cg.println("  add x0, x29, #%d", offset)
			}
		} else if paramIndex >= 8 && paramIndex <= 11 {
			offset := 16 + 8*(paramIndex-8)
			cg.emitLoad("x0", "x29", offset)
		} else {
			// ローカル変数
			if offset, exists := localVars.getOffset(e.name); exists {
				if offset < 0 {
					cg.println("  sub x0, sp, #%d", -offset)
				} else {
					cg.println("  add x0, sp, #%d", offset)
				}
			}
		}
	case *ExprOp:
		if e.op == "*" && len(e.args) == 1 {
			// ポインタ参照
			cg.genExpr(e.args[0], params, localVars)
		}
	}
}

// 値のロード
func (cg *CodeGen) load(size int, isUnsigned bool) {
	switch size {
	case 1:
		if isUnsigned {
			cg.println("  ldrb w0, [x0]")
		} else {
			cg.println("  ldrsb w0, [x0]")
		}
	case 2:
		if isUnsigned {
			cg.println("  ldrh w0, [x0]")
		} else {
			cg.println("  ldrsh w0, [x0]")
		}
	case 4:
		cg.println("  ldr w0, [x0]")
	case 8:
		cg.println("  ldr x0, [x0]")
	}
}

// 値のストア
func (cg *CodeGen) store(size int) {
	cg.pop("x1") // アドレス
	switch size {
	case 1:
		cg.println("  strb w0, [x1]")
	case 2:
		cg.println("  strh w0, [x1]")
	case 4:
		cg.println("  str w0, [x1]")
	case 8:
		cg.println("  str x0, [x1]")
	}
}

// ゼロ比較
func (cg *CodeGen) cmpZero(size int) {
	if size <= 4 {
		cg.println("  cmp w0, #0")
	} else {
		cg.println("  cmp x0, #0")
	}
}

// 型キャスト
func (cg *CodeGen) cast(fromSize, toSize int, fromUnsigned, toUnsigned bool) {
	if fromSize == toSize {
		return
	}

	if fromSize < toSize {
		// 拡張
		if fromSize == 1 {
			if fromUnsigned {
				cg.println("  uxtb w0, w0")
			} else {
				cg.println("  sxtb w0, w0")
			}
		} else if fromSize == 2 {
			if fromUnsigned {
				cg.println("  uxth w0, w0")
			} else {
				cg.println("  sxth w0, w0")
			}
		} else if fromSize == 4 && toSize == 8 {
			if fromUnsigned {
				cg.println("  uxtw x0, w0")
			} else {
				cg.println("  sxtw x0, w0")
			}
		}
	} else {
		// 縮小（上位ビットを切り捨て）
		if toSize <= 4 {
			cg.println("  and w0, w0, #0x%x", (1<<(toSize*8))-1)
		}
	}
}

// 関数呼び出しの引数処理
func (cg *CodeGen) pushArgs(args []Expr, params []string, localVars *LocalVars) int {
	stackArgs := 0
	stackArgNum := 0
	if len(args) > 8 {
		stackArgNum = len(args) - 8
		stackArgSize := alignTo(stackArgNum*8, 16)
		cg.println("  sub sp, sp, #%d", stackArgSize)
		for i := 8; i < len(args); i++ {
			cg.genExpr(args[i], params, localVars)
			offset := (i - 8) * 8
			cg.println("  str x0, [sp, #%d]", offset)
			stackArgs++
		}
	}

	// 最初の8個の引数をレジスタに配置するため、一時的にスタックに保存
	for i := 0; i < len(args) && i < 8; i++ {
		cg.genExpr(args[i], params, localVars)
		cg.push()
	}

	// レジスタ引数をスタックから取り出してレジスタに配置（逆順）
	for i := min(len(args), 8) - 1; i >= 0; i-- {
		reg := fmt.Sprintf("x%d", i)
		cg.pop(reg)
	}

	return stackArgs
}

// min関数のヘルパー
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 式のコード生成
func (cg *CodeGen) genExpr(expr Expr, params []string, localVars *LocalVars) {
	switch e := expr.(type) {
	case *ExprIntLiteral:
		// 整数リテラル
		if e.val >= 0 && e.val <= 0xFFFF {
			cg.println("  mov x0, #%d", e.val)
		} else {
			cg.println("  mov x0, #%d", e.val)
		}

	case *ExprId:
		// 変数参照
		paramIndex := getParamIndex(e.name, params)
		if paramIndex >= 0 && paramIndex <= 7 {
			total := len(params)*8 + localVars.stackSize + 256
			aligned := alignTo(total, 16)
			cg.emitLoad("x0", "x29", paramIndex*8-aligned)
		} else if paramIndex >= 8 && paramIndex <= 11 {
			// x29 はスタックフレーム基準。16 は prologue の push 分
			offset := 16 + 8*(paramIndex-8)
			cg.emitLoad("x0", "x29", offset)
		} else {
			// ローカル変数
			if offset, exists := localVars.getOffset(e.name); exists {
				// ローカル変数はパラメータ領域の後に配置
				actualOffset := len(params)*8 + offset - 8
				cg.emitLoad("x0", "sp", actualOffset)
			}
		}

	case *ExprOp:
		if len(e.args) == 1 {
			// 単項演算
			cg.genUnaryOp(e.op, e.args[0], params, localVars)
		} else if len(e.args) == 2 {
			// 二項演算
			cg.genBinaryOp(e.op, e.args[0], e.args[1], params, localVars)
		}

	case *ExprCall:
		// 関数呼び出し
		cg.genFunctionCall(e, params, localVars)
	}
}

// 単項演算のコード生成
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

// 二項演算のコード生成
func (cg *CodeGen) genBinaryOp(op string, left, right Expr, params []string, localVars *LocalVars) {
	switch op {
	case "=":
		// 代入
		cg.genExpr(right, params, localVars)
		// 左辺のアドレスを計算
		if leftId, ok := left.(*ExprId); ok {
			paramIndex := getParamIndex(leftId.name, params)
			if paramIndex >= 0 && paramIndex <= 7 {
				// パラメータの場合、代入は無視
				return
			} else {
				// ローカル変数
				if offset, exists := localVars.getOffset(leftId.name); exists {
					actualOffset := len(params)*8 + offset - 8
					cg.emitStore("x0", "sp", actualOffset)
					return
				}
			}
		}
		return

	default:
		// その他の演算
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
			// 論理AND
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
			// 論理OR
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

// 関数呼び出しのコード生成
func (cg *CodeGen) genFunctionCall(call *ExprCall, params []string, localVars *LocalVars) {
	// 引数を処理
	_ = cg.pushArgs(call.args, params, localVars)
	stackArgNum := 0
	if len(call.args) > 8 {
		stackArgNum = len(call.args) - 8
	}

	// 関数名を取得
	if funId, ok := call.fun.(*ExprId); ok {
		// 直接関数名を指定して呼び出し
		cg.println("  bl %s", funId.name)
	} else {
		// 間接呼び出し（関数ポインタなど）
		cg.genExpr(call.fun, params, localVars)
		cg.println("  blr x0")
	}

	// スタックを復元
	if stackArgNum > 0 {
		cg.println("  add sp, sp, #%d", alignTo(stackArgNum*8, 16))
	}
}

// 文のコード生成
func (cg *CodeGen) genStmt(stmt Stmt, params []string, localVars *LocalVars) {
	switch s := stmt.(type) {
	case *StmtReturn:
		if s.expr != nil {
			cg.genExpr(s.expr, params, localVars)
		}
		// スタックを復元してreturn
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

	// StmtForは現在のminCではサポートされていないため削除

	case *StmtExpr:
		cg.genExpr(s.expr, params, localVars)
	}
}

// 宣言のコード生成
func (cg *CodeGen) genDecl(decl *Decl, localVars *LocalVars) {
	// ローカル変数の宣言は既にスタックオフセットが計算済み
	// 現在のminCでは初期化はサポートされていない
}

// 関数のコード生成
func (cg *CodeGen) genFunction(fun *DefFun) {
	// パラメータ名リストを作成
	paramNames := make([]string, len(fun.params))
	for i, decl := range fun.params {
		paramNames[i] = decl.name
	}

	// ローカル変数管理を初期化
	localVars := newLocalVars()

	// ローカル変数を処理
	var localDecls []*Decl
	if body, ok := fun.body.(*StmtCompound); ok {
		localDecls = body.decls
	}

	for _, decl := range localDecls {
		localVars.addVariable(decl.name)
	}

	// 関数の開始
	cg.println(".globl %s", fun.name)
	cg.println(".type %s, @function", fun.name)
	cg.println("%s:", fun.name)

	// プロローグ
	cg.println("  stp x29, x30, [sp, #-16]!")
	cg.println("  mov x29, sp")

	// パラメータとローカル変数のためのスタック領域を確保
	// 一時的なスタック操作のためにさらに256バイト確保
	totalStackSize := len(paramNames)*8 + localVars.stackSize + 256
	alignedSize := alignTo(totalStackSize, 16)
	cg.println("  sub sp, sp, #%d", alignedSize)

	// パラメータをスタックに保存
	for i, _ := range paramNames {
		if i < 8 {
			reg := getParamRegister(i)
			offset := i * 8
			cg.println("  str %s, [sp, #%d]", reg, offset)
		}
	}

	// 関数本体を処理
	cg.genStmt(fun.body, paramNames, localVars)

	// エピローグ（return文がない場合のフォールバック）
	// 実際には、すべての関数はreturn文を持つべき
}

// プログラム全体のコード生成
func ast_to_asm_program(program *Program) string {
	if len(program.defs) == 0 {
		return ""
	}

	cg := newCodeGen()

	// データセクション
	cg.println(".data")

	// テキストセクション
	cg.println(".text")

	// 各定義を処理
	for _, def := range program.defs {
		switch d := def.(type) {
		case *DefFun:
			cg.genFunction(d)
		}
	}

	return cg.output
}

// パラメータ名からインデックスを取得
func getParamIndex(name string, params []string) int {
	for i, n := range params {
		if n == name {
			return i
		}
	}
	// a0, a1, ... のような引数名にも対応
	if len(name) >= 2 && name[0] == 'a' {
		if idx, err := strconv.Atoi(name[1:]); err == nil && idx >= 0 && idx <= 11 {
			return idx
		}
	}
	return -1
}

// パラメータインデックスからレジスタ名を取得
func getParamRegister(index int) string {
	registers := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7"}
	if index >= 0 && index < len(registers) {
		return registers[index]
	}
	return "x0"
}

// 後方互換性のための関数
func ast_to_asm_function(fun *DefFun) string {
	cg := newCodeGen()
	cg.genFunction(fun)
	return cg.output
}

func ast_to_asm_stmt(stmt Stmt, params []string, localVars *LocalVars) string {
	cg := newCodeGen()
	cg.genStmt(stmt, params, localVars)
	return cg.output
}

func ast_to_asm_expr(expr Expr, params []string, localVars *LocalVars) string {
	cg := newCodeGen()
	cg.genExpr(expr, params, localVars)
	return cg.output
}

func ast_to_asm_unary_op(op string, arg Expr, params []string, localVars *LocalVars) string {
	cg := newCodeGen()
	cg.genUnaryOp(op, arg, params, localVars)
	return cg.output
}

func ast_to_asm_binary_op(op string, left, right Expr, params []string, localVars *LocalVars) string {
	cg := newCodeGen()
	cg.genBinaryOp(op, left, right, params, localVars)
	return cg.output
}

func ast_to_asm_decl(decl *Decl, localVars *LocalVars) string {
	cg := newCodeGen()
	cg.genDecl(decl, localVars)
	return cg.output
}

func (cg *CodeGen) emitLoad(dst, base string, offset int) {
	if offset >= -256 && offset <= 255 {
		cg.println("  ldr %s, [%s, #%d]", dst, base, offset)
	} else {
		if offset < 0 {
			cg.println("  sub x9, %s, #%d", base, -offset) // x9: 作業レジスタ
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

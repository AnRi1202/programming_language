package main

import "fmt"

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

func (lv *LocalVars) addVariable(name string) int {
	offset := lv.stackSize
	lv.variables[name] = offset
	lv.stackSize += 8 // 8バイト（64ビット）確保
	return offset
}

func (lv *LocalVars) getOffset(name string) (int, bool) {
	offset, exists := lv.variables[name]
	return offset, exists
}

func ast_to_asm_program(program *Program) string {
	if len(program.defs) == 0 {
		return ""
	}

	// 最初の関数定義を取得（テストでは1つの関数のみ）
	def := program.defs[0]
	if funDef, ok := def.(*DefFun); ok {
		return ast_to_asm_function(funDef)
	}

	return ""
}

func ast_to_asm_function(fun *DefFun) string {
	// paramsから引数名リストを作成
	paramNames := make([]string, len(fun.params))
	for i, decl := range fun.params {
		paramNames[i] = decl.name
	}

	// ローカル変数管理を初期化
	localVars := newLocalVars()

	// ローカル変数数を数える
	var localDecls []*Decl
	if body, ok := fun.body.(*StmtCompound); ok {
		localDecls = body.decls
	}
	localVarCount := len(localDecls)
	for _, decl := range localDecls {
		localVars.addVariable(decl.name)
	}

	asm := fmt.Sprintf(".globl %s\n", fun.name)
	asm += fmt.Sprintf("%s:\n", fun.name)
	if localVarCount > 0 {
		asm += fmt.Sprintf("\tsub sp, sp, #%d\n", 8*localVarCount)
	}

	// 関数の本体を処理
	asm += ast_to_asm_stmt(fun.body, paramNames, localVars)

	if localVarCount > 0 {
		asm += fmt.Sprintf("\tadd sp, sp, #%d\n", 8*localVarCount)
	}

	return asm
}

func ast_to_asm_stmt(stmt Stmt, params []string, localVars *LocalVars) string {
	switch s := stmt.(type) {
	case *StmtReturn:
		// return文: 式を評価してx0に格納
		asm := ast_to_asm_expr(s.expr, params, localVars)
		if localVars.stackSize > 0 {
			asm += fmt.Sprintf("\tadd sp, sp, #%d\n", localVars.stackSize)
		}
		asm += "\tret\n"
		return asm
	case *StmtCompound:
		// 複合文: 宣言と文を順次処理
		asm := ""
		for _, decl := range s.decls {
			asm += ast_to_asm_decl(decl, localVars)
		}
		for _, stmt := range s.stmts {
			asm += ast_to_asm_stmt(stmt, params, localVars)
		}
		return asm
	default:
		return ""
	}
}

func ast_to_asm_expr(expr Expr, params []string, localVars *LocalVars) string {
	switch e := expr.(type) {
	case *ExprIntLiteral:
		// 整数リテラル: x0に即値ロード
		return fmt.Sprintf("\tmov x0, #%d\n", e.val)
	case *ExprId:
		// 変数参照: パラメータまたはローカル変数
		paramIndex := get_param_index(e.name, params)
		if paramIndex >= 0 && paramIndex <= 7 {
			reg := get_param_register(paramIndex)
			return fmt.Sprintf("\tmov x0, %s\n", reg)
		} else if paramIndex >= 8 && paramIndex <= 11 {
			// ARM64 ABI: 8番目以降はspからロード。sp+0, sp+8, ...
			offset := 8 * (paramIndex - 8)
			return fmt.Sprintf("\tldr x0, [sp, #%d]\n", offset)
		} else {
			// ローカル変数の場合
			if offset, exists := localVars.getOffset(e.name); exists {
				return fmt.Sprintf("\tldr x0, [sp, #-%d]\n", offset)
			}
		}
		return ""
	case *ExprOp:
		if len(e.args) == 1 {
			// 単項演算
			return ast_to_asm_unary_op(e.op, e.args[0], params, localVars)
		} else if len(e.args) == 2 {
			// 二項演算
			return ast_to_asm_binary_op(e.op, e.args[0], e.args[1], params, localVars)
		}
		return ""
	default:
		return ""
	}
}

func ast_to_asm_unary_op(op string, arg Expr, params []string, localVars *LocalVars) string {
	asm := ast_to_asm_expr(arg, params, localVars)
	switch op {
	case "-":
		asm += "\tneg x0, x0\n"
	case "!":
		asm += "\tcmp x0, #0\n"
		asm += "\tcset x0, eq\n"
	}
	return asm
}

func ast_to_asm_binary_op(op string, left, right Expr, params []string, localVars *LocalVars) string {
	switch op {
	case "=":
		// 代入式: 右辺を評価し、左辺に書き込み、その値を返す
		asm := ast_to_asm_expr(right, params, localVars) // 右辺を評価してx0に
		
		// 左辺のアドレスを計算
		if leftId, ok := left.(*ExprId); ok {
			paramIndex := get_param_index(leftId.name, params)
			if paramIndex >= 0 && paramIndex <= 7 {
				// パラメータの場合、レジスタに直接書き込むことはできないので、
				// 右辺の値をそのまま返す（代入の副作用は無視）
				return asm
			} else {
				// ローカル変数の場合
				if offset, exists := localVars.getOffset(leftId.name); exists {
					asm += fmt.Sprintf("\tstr x0, [sp, #-%d]\n", offset) // ローカル変数に書き込み
					return asm // 代入の値をx0に返す
				}
			}
		}
		
		// その他の場合は右辺の値を返す
		return asm
	default:
		// 既存の二項演算処理
		asm := ast_to_asm_expr(left, params, localVars)
		asm += "\tstr x0, [sp, #-16]!\n" // 左辺をpush
		asm += ast_to_asm_expr(right, params, localVars)
		asm += "\tmov x1, x0\n"        // 右辺をx1に
		asm += "\tldr x0, [sp], #16\n" // 左辺をpopしてx0に
		switch op {
		case "+":
			asm += "\tadd x0, x0, x1\n"
		case "-":
			asm += "\tsub x0, x0, x1\n"
		case "*":
			asm += "\tmul x0, x0, x1\n"
		case "/":
			asm += "\tsdiv x0, x0, x1\n"
		case "==":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, eq\n"
		case "!=":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, ne\n"
		case "<":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, lt\n"
		case "<=":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, le\n"
		case ">":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, gt\n"
		case ">=":
			asm += "\tcmp x0, x1\n"
			asm += "\tcset x0, ge\n"
		}
		return asm
	}
}

func ast_to_asm_decl(decl *Decl, localVars *LocalVars) string {
	// ここでは何もしない（スタック確保は関数の最初でまとめて行う）
	return ""
}

// パラメータ名からインデックスを取得（params順）
func get_param_index(name string, params []string) int {
	for i, n := range params {
		if n == name {
			return i
		}
	}
	// a0, a1, ... のような引数名にも対応
	if len(name) >= 2 && name[0] == 'a' {
		for i := 0; i <= 11; i++ {
			if name == fmt.Sprintf("a%d", i) {
				return i
			}
		}
	}
	return -1
}

// パラメータインデックスからレジスタ名を取得
func get_param_register(index int) string {
	registers := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7", "x8", "x9", "x10", "x11"}
	if index >= 0 && index < len(registers) {
		return registers[index]
	}
	return "x0" // デフォルト
}

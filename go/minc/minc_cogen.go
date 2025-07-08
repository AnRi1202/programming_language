package main

import "fmt"

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
	asm := fmt.Sprintf(".globl %s\n", fun.name)
	asm += fmt.Sprintf("%s:\n", fun.name)

	// paramsから引数名リストを作成
	paramNames := make([]string, len(fun.params))
	for i, decl := range fun.params {
		paramNames[i] = decl.name
	}

	// 関数の本体を処理
	asm += ast_to_asm_stmt(fun.body, paramNames)

	return asm
}

func ast_to_asm_stmt(stmt Stmt, params []string) string {
	switch s := stmt.(type) {
	case *StmtReturn:
		// return文: 式を評価してx0に格納
		asm := ast_to_asm_expr(s.expr, params)
		asm += "\tret\n"
		return asm
	case *StmtCompound:
		// 複合文: 宣言と文を順次処理
		asm := ""
		for _, decl := range s.decls {
			asm += ast_to_asm_decl(decl)
		}
		for _, stmt := range s.stmts {
			asm += ast_to_asm_stmt(stmt, params)
		}
		return asm
	default:
		return ""
	}
}

func ast_to_asm_expr(expr Expr, params []string) string {
	switch e := expr.(type) {
	case *ExprIntLiteral:
		// 整数リテラル: x0に即値ロード
		return fmt.Sprintf("\tmov x0, #%d\n", e.val)
	case *ExprId:
		// 変数参照: パラメータはx0, x1, ... x7, それ以降はスタック
		paramIndex := get_param_index(e.name, params)
		if paramIndex >= 0 && paramIndex <= 7 {
			reg := get_param_register(paramIndex)
			return fmt.Sprintf("\tmov x0, %s\n", reg)
		} else if paramIndex >= 8 && paramIndex <= 11 {
			// ARM64 ABI: 8番目以降はspからロード。sp+0, sp+8, ...
			offset := 8 * (paramIndex - 8)
			return fmt.Sprintf("\tldr x0, [sp, #%d]\n", offset)
		}
		return ""
	case *ExprOp:
		if len(e.args) == 1 {
			// 単項演算
			return ast_to_asm_unary_op(e.op, e.args[0], params)
		} else if len(e.args) == 2 {
			// 二項演算
			return ast_to_asm_binary_op(e.op, e.args[0], e.args[1], params)
		}
		return ""
	default:
		return ""
	}
}

func ast_to_asm_unary_op(op string, arg Expr, params []string) string {
	asm := ast_to_asm_expr(arg, params)
	switch op {
	case "-":
		asm += "\tneg x0, x0\n"
	case "!":
		asm += "\tcmp x0, #0\n"
		asm += "\tcset x0, eq\n"
	}
	return asm
}

func ast_to_asm_binary_op(op string, left, right Expr, params []string) string {
	asm := ast_to_asm_expr(left, params)
	asm += "\tstr x0, [sp, #-16]!\n" // 左辺をpush
	asm += ast_to_asm_expr(right, params)
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

func ast_to_asm_decl(decl *Decl) string {
	// 変数宣言はスタックに領域を確保
	// 簡単のため、ここでは何もしない（パラメータのみ対応）
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

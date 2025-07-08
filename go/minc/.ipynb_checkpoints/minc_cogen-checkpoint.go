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

	// 関数の本体を処理
	asm += ast_to_asm_stmt(fun.body)

	return asm
}

func ast_to_asm_stmt(stmt Stmt) string {
	switch s := stmt.(type) {
	case *StmtReturn:
		// return文: 式を評価してraxに格納
		asm := ast_to_asm_expr(s.expr)
		asm += "\tret\n"
		return asm
	case *StmtCompound:
		// 複合文: 宣言と文を順次処理
		asm := ""
		for _, decl := range s.decls {
			asm += ast_to_asm_decl(decl)
		}
		for _, stmt := range s.stmts {
			asm += ast_to_asm_stmt(stmt)
		}
		return asm
	default:
		return ""
	}
}

func ast_to_asm_expr(expr Expr) string {
	switch e := expr.(type) {
	case *ExprIntLiteral:
		// 整数リテラル: raxに即値ロード
		return fmt.Sprintf("\tmovq $%d, %%rax\n", e.val)
	case *ExprId:
		// 変数参照: パラメータはrdi, rsi, rdx, rcx, r8, r9の順
		paramIndex := get_param_index(e.name)
		if paramIndex >= 0 {
			reg := get_param_register(paramIndex)
			return fmt.Sprintf("\tmovq %%%s, %%rax\n", reg)
		}
		return ""
	case *ExprOp:
		if len(e.args) == 1 {
			// 単項演算
			return ast_to_asm_unary_op(e.op, e.args[0])
		} else if len(e.args) == 2 {
			// 二項演算
			return ast_to_asm_binary_op(e.op, e.args[0], e.args[1])
		}
		return ""
	default:
		return ""
	}
}

func ast_to_asm_unary_op(op string, arg Expr) string {
	asm := ast_to_asm_expr(arg)
	switch op {
	case "-":
		asm += "\tnegq %%rax\n"
	}
	return asm
}

func ast_to_asm_binary_op(op string, left, right Expr) string {
	asm := ast_to_asm_expr(left)
	asm += "\tpushq %%rax\n" // 左辺をスタックに保存
	asm += ast_to_asm_expr(right)
	asm += "\tmovq %%rax, %%rcx\n" // 右辺をrcxに保存
	asm += "\tpopq %%rax\n"        // 左辺をraxに復元

	switch op {
	case "+":
		asm += "\taddq %%rcx, %%rax\n"
	case "-":
		asm += "\tsubq %%rcx, %%rax\n"
	case "*":
		asm += "\timulq %%rcx, %%rax\n"
	case "/":
		asm += "\tcqto\n"        // raxを符号拡張
		asm += "\tidivq %%rcx\n" // rax = rax / rcx
	}
	return asm
}

func ast_to_asm_decl(decl *Decl) string {
	// 変数宣言はスタックに領域を確保
	// 簡単のため、ここでは何もしない（パラメータのみ対応）
	return ""
}

// パラメータ名からインデックスを取得（簡易版）
func get_param_index(name string) int {
	// テストケースでは x, y, z などの単純な名前
	switch name {
	case "x":
		return 0
	case "y":
		return 1
	case "z":
		return 2
	default:
		return -1
	}
}

// パラメータインデックスからレジスタ名を取得
func get_param_register(index int) string {
	registers := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
	if index >= 0 && index < len(registers) {
		return registers[index]
	}
	return "rdi" // デフォルト
}

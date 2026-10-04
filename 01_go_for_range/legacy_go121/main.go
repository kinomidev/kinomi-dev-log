// legacy_go121: go.mod の go ディレクティブを 1.21 にしたモジュールで、親の main.go と同じループを動かす。
// 実行: go run .
// go run main.go（ファイル指定）では go.mod の版が反映されず、Go 1.22 以降の仕様で動く。
package main

import (
	"fmt"
	"runtime"
)

const (
	bar  = "================================================================"
	rule = "----------------------------------------------------------------"
)

func main() {
	fmt.Println(bar)
	fmt.Println(" legacy_go121: go.mod を go 1.21 にしたモジュールの同じループ")
	fmt.Printf(" toolchain : %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf(" loop var  : %s\n", loopVarMode())
	fmt.Println(bar)

	loopVarAddrs()
	deferCapture()
}

// := で宣言したループ変数のアドレスを集め、クロージャからも読む。
// Go 1.22 以降は反復ごとに別の変数になり、旧仕様では 1 つの変数を共有する。
func loopVarAddrs() {
	section("ループ変数のアドレスとクロージャ（range と、3 つの節の for）")
	var ptrs []*int
	var funcs []func() int
	for _, v := range []int{10, 20, 30} {
		ptrs = append(ptrs, &v)
		funcs = append(funcs, func() int { return v })
	}
	for i := 0; i < 3; i++ {
		ptrs = append(ptrs, &i)
		funcs = append(funcs, func() int { return i })
	}
	fmt.Println(" loop   | n | ptr              | *ptr | closure()")
	fmt.Println("--------+---+------------------+------+----------")
	for n := range ptrs {
		loop := "range"
		if n >= 3 {
			loop = "clause"
		}
		fmt.Printf(" %-6s | %d | %-16s | %4d | %9d\n", loop, n%3, addr(ptrs[n]), *ptrs[n], funcs[n]())
	}
}

// defer で登録した関数は、関数の終わりに登録と逆の順で動く。そのとき読む v を確かめる。
// 旧仕様のファイルでは、go vet がこの書き方を警告する（loopclosure）。
func deferCapture() {
	section("defer の中から v を読む（登録と逆の順に実行される）")
	fmt.Print(" printed:")
	func() {
		for _, v := range []int{10, 20, 30} {
			defer func() { fmt.Printf(" %d", v) }()
		}
	}()
	fmt.Println()
}

// loopVarMode は、ループ変数が反復ごとに別の変数か、1 つの変数の共有かを実測する。
func loopVarMode() string {
	var ptrs []*int
	for i := 0; i < 2; i++ {
		ptrs = append(ptrs, &i)
	}
	if ptrs[0] == ptrs[1] {
		return "shared（1 つの変数を共有する: Go 1.21 以前の仕様）"
	}
	return "per-iteration（反復ごとに別の変数: Go 1.22 以降の仕様）"
}

func section(title string) {
	fmt.Printf("\n%s\n %s\n%s\n", rule, title, rule)
}

// addr はアドレスを表の列幅に合わせるため、文字列にする。
func addr(p any) string { return fmt.Sprintf("%p", p) }

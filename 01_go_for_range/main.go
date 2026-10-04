// 01_go_for_range: for range の値コピーと、ループ変数のスコープ（Go 1.22 の仕様変更）を検証する。
// 実行: go run main.go
// 旧仕様（Go 1.21 以前）との比較: legacy_go121/ で go run .
package main

import (
	"fmt"
	"runtime"
)

const (
	bar  = "================================================================"
	rule = "----------------------------------------------------------------"
)

type User struct {
	Name  string
	Score int
}

func main() {
	fmt.Println(bar)
	fmt.Println(" 01_go_for_range: for range の値コピーとループ変数の検証")
	fmt.Printf(" toolchain : %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf(" loop var  : %s\n", loopVarMode())
	fmt.Println(bar)

	ngModifyRangeValue()
	okModifyByIndex()
	ngRangeArrayByValue()
	okRangeArrayByPointer()
	loopVarAddrs()
	deferCapture()
	ngAssignToOuterVar()
	okPointToElements()
	summary()
}

// [NG] range の u は要素のコピー。u を書き換えても users は変わらない。
func ngModifyRangeValue() {
	section("[NG] range の値 u を書き換える")
	users := []User{{"alice", 10}, {"bob", 20}, {"carol", 30}}
	fmt.Println(" i | &u               | &users[i]        | u.Score")
	fmt.Println("---+------------------+------------------+--------")
	for i, u := range users {
		u.Score += 100 // コピーを書き換えているだけ
		fmt.Printf(" %d | %-16s | %-16s | %7d\n", i, addr(&u), addr(&users[i]), u.Score)
	}
	printUsers(users)
}

// [OK] インデックスで元の要素を直接書き換える。
func okModifyByIndex() {
	section("[OK] users[i] を直接書き換える")
	users := []User{{"alice", 10}, {"bob", 20}, {"carol", 30}}
	for i := range users {
		users[i].Score += 100
	}
	printUsers(users)
}

// [NG] 配列を値で range すると、ループの開始時に配列全体がコピーされる。
func ngRangeArrayByValue() {
	section("[NG] 配列 arr を値で range し、ループ中に arr を書き換える")
	arr := [3]int{1, 2, 3}
	fmt.Println(" i |   v | arr[i]")
	fmt.Println("---+-----+-------")
	for i, v := range arr {
		if i == 0 {
			arr[1], arr[2] = 200, 300 // 元の配列を書き換える
		}
		fmt.Printf(" %d | %3d | %6d\n", i, v, arr[i])
	}
}

// [OK] 配列へのポインタを range すれば、コピーせずに元の配列を読む。
func okRangeArrayByPointer() {
	section("[OK] &arr を range し、ループ中に arr を書き換える")
	arr := [3]int{1, 2, 3}
	fmt.Println(" i |   v | arr[i]")
	fmt.Println("---+-----+-------")
	for i, v := range &arr {
		if i == 0 {
			arr[1], arr[2] = 200, 300
		}
		fmt.Printf(" %d | %3d | %6d\n", i, v, arr[i])
	}
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

// [NG] ループの外で宣言した v に = で代入する形は、Go 1.22 以降も 1 つの変数を使い回す。
func ngAssignToOuterVar() {
	section("[NG] ループの外の v に = で代入し、&v を集める")
	var v int
	var ptrs []*int
	for _, v = range []int{10, 20, 30} {
		ptrs = append(ptrs, &v)
	}
	printPtrs(ptrs)
}

// [OK] 要素そのものを指したいなら、ループ変数ではなく &nums[i] を使う。
func okPointToElements() {
	section("[OK] &nums[i] で要素そのものを指す")
	nums := []int{10, 20, 30}
	var ptrs []*int
	for i := range nums {
		ptrs = append(ptrs, &nums[i])
	}
	printPtrs(ptrs)
}

func summary() {
	section("まとめ")
	fmt.Println(" 1. range の値は要素のコピー。書き換えは users[i] で行う")
	fmt.Println(" 2. 配列の range は開始時に配列ごとコピーされる。&arr で回す")
	fmt.Println(" 3. := のループ変数は反復ごとに別の変数（Go 1.22 以降）")
	fmt.Println(" 4. 外の変数に = で代入する形は、今も 1 つの変数を共有する")
	fmt.Println(" ※ アドレスは実行のたびに変わる。旧仕様の結果は legacy_go121/ を参照")
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

func printUsers(users []User) {
	fmt.Print(" -> users after loop:")
	for _, u := range users {
		fmt.Printf(" %s=%d", u.Name, u.Score)
	}
	fmt.Println()
}

func printPtrs(ptrs []*int) {
	fmt.Println(" n | ptr              | *ptr")
	fmt.Println("---+------------------+-----")
	for n, p := range ptrs {
		fmt.Printf(" %d | %-16s | %4d\n", n, addr(p), *p)
	}
}

// addr はアドレスを表の列幅に合わせるため、文字列にする。
func addr(p any) string { return fmt.Sprintf("%p", p) }

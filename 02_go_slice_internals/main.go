// 02_go_slice_internals: スライスの len と cap、append で配列が作り直される境界と、
// そのときのアドレスの変化を確かめる。
package main

import (
	"fmt"
	"runtime"
	"slices"
)

// arr は、スライスが指す配列の先頭アドレスを返す（%p はスライスの 0 番目の要素のアドレスを出す）。
func arr(s []int) string {
	return fmt.Sprintf("%p", s)
}

// header は、ケースの区切り線と見出しを出す。
func header(title string) {
	fmt.Println()
	fmt.Println("============================================================")
	fmt.Println(title)
	fmt.Println("============================================================")
}

// ---------------------------------------------------------------
// ケース 1: make の len と cap

// ngMakeLen は、make で len を 5 にしてから append する。
func ngMakeLen() {
	fmt.Println("[NG] make([]int, 5) に append する")
	s := make([]int, 5)
	for i := 1; i <= 3; i++ {
		s = append(s, i)
	}
	fmt.Printf(" s = %v  len=%d cap=%d\n", s, len(s), cap(s))
}

// okMakeCap は、len を 0、cap を 5 にしてから append する。
func okMakeCap() {
	fmt.Println("[OK] make([]int, 0, 5) に append する")
	s := make([]int, 0, 5)
	for i := 1; i <= 3; i++ {
		s = append(s, i)
	}
	fmt.Printf(" s = %v  len=%d cap=%d\n", s, len(s), cap(s))
}

// ---------------------------------------------------------------
// ケース 2: append で配列が作り直される境界

// observeBoundary は、nil のスライスに 1 個ずつ append し、len、cap、配列のアドレスを表にする。
func observeBoundary() {
	fmt.Println("[観察] nil のスライスに 1 個ずつ append する")
	fmt.Println(" len | cap | array            | moved")
	fmt.Println("-----+-----+------------------+------")
	var s []int
	prev := ""
	for i := 1; i <= 10; i++ {
		s = append(s, i)
		moved := "no"
		switch {
		case prev == "":
			moved = "new"
		case arr(s) != prev:
			moved = "yes"
		}
		fmt.Printf(" %3d | %3d | %-16s | %s\n", len(s), cap(s), arr(s), moved)
		prev = arr(s)
	}
}

// ngStalePointer は、append の前に取った要素のポインタを、append のあとも使う。
func ngStalePointer() {
	fmt.Println("[NG] append の前に取ったポインタで書き換える")
	s := []int{10, 20, 30} // len 3, cap 3
	p := &s[0]
	s = append(s, 40) // cap を超えるので、新しい配列に移る
	*p = 999          // 古い配列を書き換えている
	fmt.Printf(" s = %v  *p = %d  p == &s[0]: %t\n", s, *p, p == &s[0])
}

// okRetakePointer は、append のあとでポインタを取り直す。
func okRetakePointer() {
	fmt.Println("[OK] append のあとでポインタを取り直す")
	s := []int{10, 20, 30}
	s = append(s, 40)
	p := &s[0]
	*p = 999
	fmt.Printf(" s = %v  *p = %d  p == &s[0]: %t\n", s, *p, p == &s[0])
}

// ---------------------------------------------------------------
// ケース 3: 同じ配列を指すスライスへの append

// ngAppendSubslice は、部分スライスに append して、元のスライスの要素を上書きする。
func ngAppendSubslice() {
	fmt.Println("[NG] b := a[:2] に append する")
	a := []int{1, 2, 3, 4, 5}
	b := a[:2] // len 2, cap 5。a と同じ配列を指す
	b = append(b, 99)
	fmt.Printf(" a = %v\n b = %v  len=%d cap=%d  same array: %t\n", a, b, len(b), cap(b), arr(a) == arr(b))
}

// okFullSliceExpr は、3 つ目の添字で cap を len と同じ 2 にしてから append する。
func okFullSliceExpr() {
	fmt.Println("[OK] b := a[:2:2] に append する")
	a := []int{1, 2, 3, 4, 5}
	b := a[:2:2] // len 2, cap 2
	b = append(b, 99)
	fmt.Printf(" a = %v\n b = %v  len=%d cap=%d  same array: %t\n", a, b, len(b), cap(b), arr(a) == arr(b))
}

// ngTwoAppends は、cap に余りのある同じ base から、append で x と y を作る。
func ngTwoAppends() {
	fmt.Println("[NG] 同じ base から x と y を append で作る")
	base := make([]int, 3, 10)
	x := append(base, 1)
	y := append(base, 2) // x と同じ配列の、同じ位置に書く
	fmt.Printf(" x = %v\n y = %v\n", x, y)
}

// okClipBeforeAppend は、slices.Clip で cap の余りを切ってから append する。
func okClipBeforeAppend() {
	fmt.Println("[OK] slices.Clip(base) に append する")
	base := make([]int, 3, 10)
	x := append(slices.Clip(base), 1)
	y := append(slices.Clip(base), 2)
	fmt.Printf(" x = %v\n y = %v\n", x, y)
}

// ---------------------------------------------------------------
// ケース 4: 関数の中の append

// appendNoReturn は、引数のスライスに append するが、結果を返さない。
func appendNoReturn(s []int, v int) {
	s = append(s, v)
}

// appendAndReturn は、append した結果を返す。
func appendAndReturn(s []int, v int) []int {
	return append(s, v)
}

// ngAppendInFunc は、関数の中で append して、結果を受け取らない。
func ngAppendInFunc() {
	fmt.Println("[NG] 関数の中で append し、結果を返さない")
	s := make([]int, 2, 4)
	appendNoReturn(s, 7)
	fmt.Printf(" s = %v  len=%d cap=%d  s[:cap(s)] = %v\n", s, len(s), cap(s), s[:cap(s)])
}

// okAppendInFunc は、append した結果を戻り値で受け取る。
func okAppendInFunc() {
	fmt.Println("[OK] append した結果を返し、呼び出し元で受け取る")
	s := make([]int, 2, 4)
	s = appendAndReturn(s, 7)
	fmt.Printf(" s = %v  len=%d cap=%d\n", s, len(s), cap(s))
}

// ---------------------------------------------------------------
// ケース 5: cap の伸び方

// observeGrowth は、nil のスライスに 1 個ずつ append し、cap が変わった行だけを出す。
func observeGrowth() {
	fmt.Println("[観察] []int に 1 個ずつ 2000 個まで append し、cap が変わった行だけ出す")
	fmt.Println(" len  | old cap | new cap | ratio | array")
	fmt.Println("------+---------+---------+-------+-----------------")
	var s []int
	for i := 0; i < 2000; i++ {
		oldCap := cap(s)
		s = append(s, i)
		if cap(s) == oldCap {
			continue
		}
		ratio := "    -"
		if oldCap > 0 {
			ratio = fmt.Sprintf("%5.2f", float64(cap(s))/float64(oldCap))
		}
		fmt.Printf(" %4d | %7d | %7d | %s | %s\n", len(s), oldCap, cap(s), ratio, arr(s))
	}
}

// sinkInts と sinkBytes は、スライスを関数の外に出す（エスケープさせる）ための入れ物。
var (
	sinkInts  [][]int
	sinkBytes [][]byte
)

// capsEscape は、append の結果をパッケージ変数に入れて関数の外に出し、cap を返す。
func capsEscape() (one, five, oneByte int) {
	a := append([]int(nil), 1)
	b := append([]int(nil), 1, 2, 3, 4, 5)
	c := append([]byte(nil), 'a')
	sinkInts = append(sinkInts, a, b)
	sinkBytes = append(sinkBytes, c)
	return cap(a), cap(b), cap(c)
}

// capsLocal は、append の結果を関数の外に出さずに、cap だけを返す。
func capsLocal() (one, five, oneByte int) {
	a := append([]int(nil), 1)
	b := append([]int(nil), 1, 2, 3, 4, 5)
	c := append([]byte(nil), 'a')
	return cap(a), cap(b), cap(c)
}

// observeSizeClass は、1 回の append で作られる cap を、スライスを外に出す場合と出さない場合で比べる。
func observeSizeClass() {
	fmt.Println("[観察] 1 回の append で作られる cap（escape: 関数の外に出す / local: 出さない）")
	fmt.Println(" expr                              | len | escape | local")
	fmt.Println("-----------------------------------+-----+--------+------")
	e1, e5, eb := capsEscape()
	l1, l5, lb := capsLocal()
	fmt.Printf(" %-33s | %3d | %6d | %5d\n", "append([]int(nil), 1)", 1, e1, l1)
	fmt.Printf(" %-33s | %3d | %6d | %5d\n", "append([]int(nil), 1, 2, 3, 4, 5)", 5, e5, l5)
	fmt.Printf(" %-33s | %3d | %6d | %5d\n", "append([]byte(nil), 'a')", 1, eb, lb)
}

// printLenCap は、len と cap だけを Println する（s は Println に渡らず、関数の外に出ない）。
func printLenCap() {
	fmt.Println("[観察] fmt.Println(len(s), cap(s)) で出す")
	var s []int
	for i := 1; i <= 5; i++ {
		s = append(s, i)
		fmt.Println(" len", len(s), "cap", cap(s))
	}
}

// printSliceToo は、s 自体も Println に渡す（s が関数の外に出る）。
func printSliceToo() {
	fmt.Println("[観察] fmt.Println(s, len(s), cap(s)) で出す")
	var s []int
	for i := 1; i <= 5; i++ {
		s = append(s, i)
		fmt.Println(" s", s, "len", len(s), "cap", cap(s))
	}
}

// summary は、ログからわかったことをまとめる。
func summary() {
	header("まとめ")
	fmt.Println(" 1. append は、追加後の len が cap を超えるときだけ新しい配列を作る（アドレスが変わる）")
	fmt.Println(" 2. cap に余りがあると、append は元の配列に書き込む（a[:2] や、同じ base から作ったスライスが上書きされる）")
	fmt.Println(" 3. 余りは a[:2:2] か slices.Clip で切る。append の結果は必ず受け取り、ポインタは append のあとで取り直す")
	fmt.Println(" 4. cap は 512 までは 2 倍ずつ伸び、その先は伸び率が下がる。メモリの区分に切り上がり、スライスが関数の外に出るかでも変わる")
}

func main() {
	fmt.Printf(" go: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	header("ケース 1: make の len と cap")
	ngMakeLen()
	fmt.Println()
	okMakeCap()

	header("ケース 2: append で配列が作り直される境界")
	observeBoundary()
	fmt.Println()
	ngStalePointer()
	fmt.Println()
	okRetakePointer()

	header("ケース 3: 同じ配列を指すスライスへの append")
	ngAppendSubslice()
	fmt.Println()
	okFullSliceExpr()
	fmt.Println()
	ngTwoAppends()
	fmt.Println()
	okClipBeforeAppend()

	header("ケース 4: 関数の中の append")
	ngAppendInFunc()
	fmt.Println()
	okAppendInFunc()

	header("ケース 5: cap の伸び方")
	observeGrowth()
	fmt.Println()
	observeSizeClass()
	fmt.Println()
	printLenCap()
	fmt.Println()
	printSliceToo()

	summary()
}

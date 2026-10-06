# 02_go_slice_internals

- カテゴリー: tech / 作成日: 2026-10-07
- 扱う内容: スライスの cap と len の内部挙動、append 時に配列が再確保される境界とポインタの変化
- 前提: Go 1.27.1（darwin/arm64）、標準ライブラリのみ（fmt、runtime、slices）。go.mod は `go 1.27.1`。スライスの基本的な書き方を知っている人向け

## 確かめる問い

- `make([]int, 5)` と `make([]int, 0, 5)` に append すると、結果はどう違うか
- append は、どの時点で新しい配列を作るか（配列のアドレスはいつ変わるか）
- append の前に取った要素のポインタは、append のあと何を指すか
- 部分スライス `a[:2]` に append すると、元の `a` はどうなるか。`a[:2:2]` と `slices.Clip` で防げるか
- 関数の中で append した結果は、呼び出し元から見えるか
- cap はどう伸びるか（何倍か、どこで変わるか）
- cap の値は、スライスが関数の外に出るかどうかで変わるか

## 一次情報

| # | 種別 | 出典（名称、節、URL） | 確認したこと | 確認日 |
|---|---|---|---|---|
| 1 | 言語仕様 | The Go Programming Language Specification（Language version go1.27）「Slice types」https://go.dev/ref/spec#Slice_types | スライスは、配列や同じ配列のほかのスライスと記憶域を共有する。cap は、スライスの len と、スライスの先にある配列の長さの和 | 2026-10-07 |
| 2 | 言語仕様 | 同「Appending to and copying slices」https://go.dev/ref/spec#Appending_and_copying_slices | cap が足りないときだけ、新しい十分な大きさの配列を確保する。足りるときは元の配列を使い回す | 2026-10-07 |
| 3 | 言語仕様 | 同「Slice expressions」の「Full slice expressions」https://go.dev/ref/spec#Slice_expressions | `a[low : high : max]` の cap は `max - low` | 2026-10-07 |
| 4 | 言語仕様 | 同「Making slices, maps and channels」https://go.dev/ref/spec#Making_slices_maps_and_channels | `make(T, n)` は len も cap も n。`make(T, n, m)` は len が n、cap が m | 2026-10-07 |
| 5 | 標準ライブラリ | `go doc builtin.append` | 容量が足りなければ新しい配列を確保する。append の結果は必ず受け取る必要がある | 2026-10-07 |
| 6 | 標準ライブラリ | `go doc fmt` の %p | スライスの %p は、0 番目の要素のアドレス | 2026-10-07 |
| 7 | 標準ライブラリ | `go doc slices.Clip`、`go doc slices.Grow` | Clip は `s[:len(s):len(s)]` を返す。Grow(n) のあとは、n 個を再確保なしで追加できる | 2026-10-07 |
| 8 | 実装 | Go 1.27.1 の `src/runtime/slice.go`（`type slice`、`growslice`、`nextslicecap`） | スライスの中身は配列へのポインタ、len、cap。新しい len が今の cap の 2 倍を超えればその len、cap が 256 未満なら 2 倍、256 以上なら `newcap += (newcap + 3*256) >> 2` を足りるまで繰り返す。そのあと `roundupsize` で切り上げる | 2026-10-07 |
| 9 | 実装 | Go 1.27.1 の `src/runtime/msize.go`（`roundupsize`）、`src/internal/runtime/gc/sizeclasses.go`（`SizeClassToSize`） | 確保するバイト数を、メモリの区分（8、16、24、32、48、…、6784、…、10240、…、14336、…、20480 バイトなど）に切り上げる | 2026-10-07 |
| 10 | 実装 | Go 1.27.1 の `src/cmd/compile/internal/ssagen/ssa.go`（3925 行目のコメント、`getBackingStoreInfo`）、`src/cmd/compile/internal/base/flag.go`（`VariableMakeThreshold = 32`） | append の結果が関数の外に出ないとき、追加後の len が K（32 バイト ÷ 要素の大きさ）以下なら、最初の append でスタック上の配列（K 個分）を使う | 2026-10-07 |

## 実行・実践の記録

```text
go -C 02_go_slice_internals vet ./...                 -> 警告なし（exit 0）
go -C 02_go_slice_internals run main.go               -> output.txt（exit 0）
go -C 02_go_slice_internals build -gcflags=-m -o /dev/null main.go
  -> capsEscape の append（190〜192 行目）: escapes to heap
  -> capsLocal の append（200〜202 行目。呼び出し元 212 行目にインライン展開）: does not escape
  -> printLenCap の append（223 行目）: does not escape
  -> printSliceToo の append（233 行目）: escapes to heap、s escapes to heap（234 行目）
go -C 02_go_slice_internals run -gcflags=-d=variablemakethreshold=0 main.go
  -> local の列が escape の列と同じ 1 / 6 / 8 になる。printLenCap の cap も 1, 2, 4, 4, 8 になる
```

## 判明したこと

- `make([]int, 5)` に 1、2、3 を append すると `[0 0 0 0 0 1 2 3]`（len 8、cap 10）。`make([]int, 0, 5)` なら `[1 2 3]`（len 3、cap 5）（根拠: output.txt ケース 1、一次情報 #4）
- 配列のアドレスが変わるのは、cap が変わった append だけ（len が 1、2、3、5、9 になったとき）。cap に余りがあるときは変わらない（根拠: output.txt ケース 2、一次情報 #2）
- append の前に取った `&s[0]` は、cap を超えた append のあとは古い配列を指す（`p == &s[0]` が false、`s[0]` は 10 のまま）（根拠: output.txt ケース 2）
- `b := a[:2]` に 99 を append すると、`a` は `[1 2 99 4 5]` になる。`a[:2:2]` なら `a` は変わらず、`b` は新しい配列（cap 4）に移る（根拠: output.txt ケース 3、一次情報 #1、#3）
- cap 10 の同じ `base` から `x := append(base, 1)`、`y := append(base, 2)` と作ると、`x` も `[0 0 0 2]` になる。`slices.Clip(base)` に append すれば `x = [0 0 0 1]`（根拠: output.txt ケース 3、一次情報 #7）
- 関数の中で append して結果を返さないと、呼び出し元の len は 2 のまま。ただし `s[:cap(s)]` で見ると、配列の 3 番目には 7 が書かれている（根拠: output.txt ケース 4、一次情報 #5、#8）
- cap は 1、2、4、8、…、256、512 までは 2 倍。その先は 848（1.66 倍）、1280（1.51 倍）、1792（1.40 倍）、2560（1.43 倍）（根拠: output.txt ケース 5）
- 512 → 848 の計算: 512 + (512 + 768) / 4 = 832 個 = 6656 バイトを、区分 6784 バイトに切り上げて 848 個。848 → 1280、1280 → 1792、1792 → 2560 も同じ計算で一致する（根拠: 一次情報 #8、#9）
- スライスを外に出すとき、`append([]int(nil), 1, 2, 3, 4, 5)` の cap は 6（40 バイト → 48 バイト）、`append([]byte(nil), 'a')` の cap は 8（1 バイト → 8 バイト）（根拠: output.txt ケース 5、一次情報 #9）
- スライスを外に出さないと、`append([]int(nil), 1)` の cap は 4（32 ÷ 8）、`append([]byte(nil), 'a')` の cap は 32（32 ÷ 1）。`fmt.Println(len(s), cap(s))` だけで出すと cap は 4、4、4、4、8 で、`s` も渡すと 1、2、4、4、8（根拠: output.txt ケース 5、一次情報 #10、`-d=variablemakethreshold=0` の結果）

## 要確認

- スタック上の配列を使う最適化（一次情報 #10）が入った Go のバージョンと、リリースノートでの扱いは、公式の情報で確かめていない（Go 1.27.1 のソースと実行結果だけで確かめた）
- cap の伸び方とスタックの配列は実装の都合で、言語仕様は保証しない。ほかのバージョンや、int が 4 バイトの 32 ビット環境では値が変わりうる（darwin/arm64 でだけ確かめた）

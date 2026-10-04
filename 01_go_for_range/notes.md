# 01_go_for_range

- カテゴリー: tech / 作成日: 2026-10-04
- 扱う内容: Goのfor rangeにおける値コピー挙動と、ループ変数による罠（Go 1.22のスコープ仕様変更との比較検証）
- 前提: go1.27.1 darwin/arm64、標準ライブラリのみ。本体のモジュールは go.mod が `go 1.27.1`、比較用の `legacy_go121/` は `go 1.21`

## 確かめる問い

1. range で取り出した値を書き換えると、元のスライスは変わるか
2. 配列を range している途中で元の配列を書き換えると、ループ変数に反映されるか
3. ループ変数のアドレスやクロージャは、Go 1.22 以降と旧仕様でどう変わるか
4. Go 1.22 以降でも、ループ変数を共有してしまう書き方は残っているか
5. 手元の Go 1.27.1 で旧仕様を再現するには、どうすればよいか

## 一次情報

| # | 種別 | 出典（名称、節、URL） | 確認したこと | 確認日 |
|---|---|---|---|---|
| 1 | 言語仕様 | The Go Programming Language Specification（Language version go1.27）「For statements with for clause」 https://go.dev/ref/spec#For_clause | 反復ごとに別の変数を持つ [Go 1.22]。次の反復の変数は post 文の前に宣言され、その時点の前の反復の値で初期化される | 2026-10-04 |
| 2 | 言語仕様 | 同「For statements with range clause」 https://go.dev/ref/spec#For_range | range 式はループ開始前に評価される。`:=` で宣言した反復変数は反復ごとに新しい変数 [Go 1.22]。range 句で宣言しない場合は既存の変数に代入文と同じように代入される | 2026-10-04 |
| 3 | 言語仕様 | 同「Language versions」 https://go.dev/ref/spec#Language_versions | Go 1.22: 反復ごとの変数、整数の range。Go 1.23: イテレータ関数の range | 2026-10-04 |
| 4 | コンパイラのソース | go1.27.1 の `src/cmd/compile/internal/noder/writer.go`（`distinctVars`） | ファイルの言語バージョンが go1.22 以上（または未設定）なら、反復ごとの変数にする | 2026-10-04 |
| 5 | コンパイラのソース | go1.27.1 の `src/cmd/compile/internal/loopvar/loopvar.go` のコメント | `GOEXPERIMENT=loopvar` は全パッケージで新しい挙動を有効にする。writer.go のコメントでは Go 1.21 も同じ扱い | 2026-10-04 |
| 6 | 実行ログ | `output.txt`、`legacy_go121/output.txt` | 問い 1〜4 の結果 | 2026-10-04 |

仕様書はローカルの `$(go env GOROOT)/doc/go_spec.html` で読み、節の id から URL を作った。

## 実行・実践の記録

```text
go -C 01_go_for_range run main.go               -> output.txt（exit 0）
go -C 01_go_for_range/legacy_go121 run .        -> legacy_go121/output.txt（exit 0）
go -C 01_go_for_range/legacy_go121 run main.go  -> loop var: per-iteration（go.mod の go 1.21 が効かない）
go build -a -n main.go / . （legacy_go121）     -> -lang=go1.27 / -lang=go1.21
go -C 01_go_for_range/legacy_go121 vet .        -> main.go:59:37: loop variable v captured by func literal（exit 1）
go -C 01_go_for_range vet ./...                 -> 警告なし（exit 0）
go -C 01_go_for_range build -gcflags=-m -o /dev/null main.go
  -> ./main.go:45:9: moved to heap: u / ./main.go:96:9: moved to heap: v
     ./main.go:100:6: moved to heap: i / ./main.go:131:6: moved to heap: v
go -C 01_go_for_range build -gcflags=-d=loopvar=2 -o /dev/null main.go
  -> ./main.go:45:9: loop variable u now per-iteration, heap-allocated
     ./main.go:96:9: loop variable v now per-iteration, heap-allocated
     ./main.go:100:6: loop variable i now per-iteration, heap-allocated
     ./main.go:121:10: loop variable v now per-iteration, stack-allocated
     （133 行目の `for _, v = range` はログに出ない）
```

legacy_go121 を `go run main.go` で実行したときの出力（抜粋）:

```text
 loop var  : per-iteration（反復ごとに別の変数: Go 1.22 以降の仕様）
 range  | 0 | 0x270369be20d8   |   10 |        10
 range  | 1 | 0x270369be20e0   |   20 |        20
 range  | 2 | 0x270369be20e8   |   30 |        30
 printed: 30 20 10
```

## 判明したこと

1. range の `u` は要素のコピー。`&u` と `&users[i]` は別のアドレスで、`u` を書き換えても `users` は変わらない（根拠: output.txt）。
2. 配列を値で range すると、ループ開始時の配列の値で回る。途中で書き換えた 200、300 は `v` に出ない。`&arr` で回すと出る（根拠: output.txt、一次情報 #2）。
3. 本体（Go 1.22 以降の仕様）では、range の `v` も 3 つの節の `i` も反復ごとに別のアドレスになり、クロージャも各反復の値を返す（根拠: output.txt、一次情報 #1、#2）。
4. 旧仕様（legacy_go121 を `go run .`）では 1 つの変数を共有する。アドレスはすべて同じで、値は最後の値（30、3）になる（根拠: legacy_go121/output.txt）。
5. Go 1.22 以降でも、外で宣言した `v` に `=` で代入する形は 1 つの変数を共有する。loopvar のログにも出ない（根拠: output.txt、一次情報 #2）。
6. `go run main.go`（ファイル指定）は、go.mod の go ディレクティブではなくツールチェーンの版（go1.27）を `-lang` に渡す。旧仕様の再現には `go run .`（パッケージ指定）が要る（根拠: `go build -n` の出力）。
7. go vet の loopclosure は、言語バージョンが 1.22 未満のファイルでだけ警告した（根拠: vet の出力）。
8. 反復ごとの変数は、アドレスが外に出ると反復ごとにヒープに確保される。外に出なければスタックに置かれる（根拠: `-gcflags=-m` と `-d=loopvar=2` の出力）。

## 要確認

- 判明したこと 6（ファイル指定の go run が go.mod の版を `-lang` に渡さない）が意図された仕様か既知の問題かは、公式の issue などで確かめていない（Web は使っていない）。

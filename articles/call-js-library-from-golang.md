---
title: "GoからJavaScriptのライブラリを呼び出す"
emoji: "🔔"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go", "javascript"]
published: true
---

## はじめに

様々な事情で JavaScript のライブラリを Go から呼び出したい場合があります。
この記事ではそれを実現する方法の一例を紹介します。

例として以下のようなシチュエーションを想像します。

- フロントエンドが JavaScript、バックエンドが Go で書かれた Web アプリケーションを開発運用している
- このアプリケーションにはユーザが文言をカスタマイズして、他のユーザにメールを送るような機能がある
- 文言はテンプレートエンジン ([nunjucks](http://mozilla.github.io/nunjucks)) を使ってカスタマイズできる
- 現在の実装はフロントエンドでテンプレートをレンダリングしている
- この機能を改良してテンプレートの変数としてサーバサイドの値を使えるようにしたい

現在の実装はこんな感じです。

```js
const someClientValue = "some client value";

async function callSendMailAPI(body) {
  await fetch("http://localhost:8192/", { method: "POST", body });
}

function fillTemplate(template) {
  return nunjucks.renderString(template, { clientValue: someClientValue });
}

function App() {
  const [template, setTemplate] = useState("<h1>Hello, {{ clientValue.toUpperCase() }}!</h1>");

  return (
    <div className="App">
      <textarea onChange={(e) => setTemplate(e.target.value)}>{template}</textarea>
      <button onClick={() => callSendMailAPI(fillTemplate(template))}>Send mail</button>
    </div>
  );
}
```

```go
// これをメールに含めたい
const someServerValue = "some server value"

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    var body bytes.Buffer
    _, _ = io.Copy(&body, r.Body)

    sendMail(body.String())

    fmt.Fprintf(w, "OK")
  })

  log.Fatal(http.ListenAndServe(":8192", nil))
}

func sendMail(content string) {
  // サンプルなので実際にはメールを送信せずログに内容を出力するだけ
  log.Printf("send mail: %s", content)
}
```

## 対応方針

サーバサイドの変数をテンプレートに埋め込むとなると、フロントエンドでテンプレートをレンダリングするというのは無理がありそうです。
サーバサイドでテンプレートをレンダリングすることにしましょう。

テンプレートエンジンには `nunjucks` が使われているということなので、Go で書かれた `nunjucks` のライブラリを探します。
しかし、 GitHub で検索してもそのようなライブラリは見当たりません。
<https://github.com/topics/go?q=nunjucks>

アプリケーションを 1 から作るなら Go 標準ライブラリの `html/template` や [pongo2](github.com/flosch/pongo2) などを使うのが良さそうです。
しかし今回は既存のアプリケーションを改修するので既存のテンプレートをそのまま使いたいです。

そこで [goja](https://github.com/dop251/goja) を使います。
`goja` は Go で書かれた JavaScript ランタイムです。
これを使って `nunjucks` のテンプレートと JavaScript のライブラリをそのまま使うことにします。

## 実装

`goja` を使うと以下のように Go コードに JavaScript を埋め込んで実行できます。

```go
vm := goja.New()
v, _ := vm.RunString(`"foo".toUpperCase()`)
fmt.Printf("%v\n", v) // FOO と出力される
```

Go には `strings.ToUpper` という関数がありますが、 `string` に `toUpperCase` というメソッドがあるわけはありません。
しかし JavaScript には `String` に `toUpperCase` というメソッドがあります。上記のサンプルはこれを利用しています。

`vm.RunString` の引数を `"foo".toUppserCase()` から `nunjucks.renderString(template, { clientValue: someClientValue })` に変えたら、それですぐに `nunjucks` を Go から使えるようになるかというとそうではありません。
`goja` が `nunjuncks` のライブラリ自体のコードを読み込んでいないためです。

どうにかして `goja` に `nunjucks` のコードも読み込ませるための手段として、 `esbuild` を使って必要なコードを全てバンドルして、それを `goja` に読み込ませることにします。

まず、 `index.js` を以下のように実装します。

```js
const nunjucks = require("nunjucks");
result = nunjucks.renderString(template, { serverValue });
```

そして以下のコマンドで `index.js` と `nunjucks` のコードを `dist.js` にバンドルします。
(`goja` は ES5 までの構文しかサポートしていないため、 `--target` には `esnext` ではなく `es2017` を指定しています)

```sh
esbuild index.js --bundle --minify --target=es2017 >dist.js
```

バンドルしたコードを `go:embed` で Go の実行ファイルに埋め込んで `goja` に読み込ませます。

```go
//go:embed dist.js
var gojaJS string

const someServerValue = "some server value"

func fillTemplate(template string) string {
   vm := goja.New()

  // JavaScript ランタイム上のグローバル変数に Go の値を渡す
   _ = vm.Set("template", template)
   _ = vm.Set("serverValue", someServerValue)

   _, _ = vm.RunString(gojaJS)

  // JavaScript ランタイム上のグローバル変数の値を読み取る
  return vm.Get("result").String()
}
```

こうすることで無事 JavaScript の `nunjucks` ライブラリを Go から呼び出すことに成功しました。

最初に書いた実装は以下のように変わります。

```diff
- const someClientValue = "some client value";
- 
async function callSendMailAPI(body) {
  await fetch("http://localhost:8192/", { method: "POST", body });
}

- function fillTemplate(template) {
-   return nunjucks.renderString(template, { clientValue: someClientValue });
- }
- 
function App() {
-  const [template, setTemplate] = useState("<h1>Hello, {{ clientValue.toUpperCase() }}!</h1>");
+  const [template, setTemplate] = useState("<h1>Hello, {{ serverValue.toUpperCase() }}!</h1>");

  return (
    <div className="App">
      <textarea onChange={(e) => setTemplate(e.target.value)}>{template}</textarea>
-      <button onClick={() => callSendMailAPI(fillTemplate(template))}>Send mail</button>
+      <button onClick={() => callSendMailAPI(template)}>Send mail</button>
    </div>
  );
}
```

```diff
+ //go:embed dist.js
+ var gojaJS string
+ 
// これをメールに含めたい
const someServerValue = "some server value"

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    var body bytes.Buffer
    _, _ = io.Copy(&body, r.Body)

-    sendMail(body.String())
+    sendMail(fillTemplate(body.String()))

    fmt.Fprintf(w, "OK")
  })

  log.Fatal(http.ListenAndServe(":8192", nil))
}

+ func fillTemplate(template string) string {
+    vm := goja.New()
+ 
+   // JavaScript ランタイム上のグローバル変数に Go の値を渡す
+    _ = vm.Set("template", template)
+    _ = vm.Set("serverValue", someServerValue)
+ 
+    _, _ = vm.RunString(gojaJS)
+ 
+   // JavaScript ランタイム上のグローバル変数の値を読み取る
+   return vm.Get("result").String()
+ }
+ 
```

## 最後に

やや強引な例かもしれませんが、どうしようもない理由で Go から JavaScript を呼び出す必要性が発生した場合の参考になれば幸いです。
また、サンプルコードとしてエラーハンドリングやバリデーションの類などはサボっていることをご了承ください。

今回は [goja](https://github.com/dop251/goja) を使用しましたが、 似たようなことを実現するライブラリとして [otto](https://github.com/robertkrimen/otto), [v8go](https://github.com/rogchap/v8go) というものがあります。
必要に応じて比較検討していただければと思います。

実行可能なサンプルコードは [GitHub](https://github.com/lambdasawa/zenn/tree/main/snippet/call-js-library-from-golang) にあるので必要に応じて参照してください。

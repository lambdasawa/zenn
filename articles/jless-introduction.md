---
title: "JSON を見る際の便利コマンド jless の紹介"
emoji: "🌙" # jless のマスコットキャラがクラゲ(海月)なので
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["json", "cli", "jq"]
published: true
---

JSON を見る際の便利コマンド `jless` の紹介をします。
一言で紹介すると JSON に特化した `less` コマンドです。

公式サイトのリンクはこちらになります。

- [インストール方法](https://github.com/PaulJuliusMartinez/jless#installation)
- [ユーザガイド](https://jless.io/user-guide.html)

## シンタックスハイライト

![less and jless](https://storage.googleapis.com/zenn-user-upload/e3594da83c76-20220723.png)

こちらが [OpenAPI スキーマの example としてよく使われる JSON](https://github.com/OAI/OpenAPI-Specification/blob/main/examples/v3.0/petstore.json) を `less` (左) と `jless` (右) にパイプで渡した際の画像です。

右はシンタックスハイライトされており視認性が高いです。

## JSON の構造を意識したカーソル移動

```txt
  "info": {
    "version": "1.0.0",
    "title": "Swagger Petstore",
    "license": {
      "name": "MIT"    <-- ①
    }    <-- ②
  },
  "servers": [    <-- ③
    {
      "url": "http://petstore.swagger.io/v1"
    }
  ],
```

① の行にカーソルがあたっている状態で `↓` キーをタイプすると、一般的なテキストエディタなどでは ② の位置にカーソルが移動します。
この操作をした場合 `jless` では ③ の位置にカーソルが移動します。

② の位置に移動したい理由はほとんどないので、 `jless` の方が簡潔に操作できると言えます。

`↑`, `↓` のキーの代わりに `j`, `k` を使うこともできます。

## 畳み込み

ある要素を探しているときに、他の興味のない要素がノイズにならないように隠したいシーンがあると思います。

例えば VSCode では行番号のあたりに `▼` が表示されるので、これをクリックして畳み込みをしたり展開したりできます。

`jless` の場合は `←` or `h` キーで畳み込み、 `→` or `l` キーで展開できます。
タイプしやすいキーに頻繁に使う操作が割り当てられているのは便利です。

## 検索

`less` と同様に `/` キーで検索を行えます。

## 要素のコピペ

以下のような JSON を `jless` で開いており ① の部分にカーソルがあたっているとします。

```txt
  "components": {
    "schemas": {
      "Pet": {    <-- ①
        "type": "object",
        "required": [
          "id",
          "name"
        ],
        "properties": {
          "id": {
            "type": "integer",
            "format": "int64"
          },
          "name": {
            "type": "string"
          },
          "tag": {
            "type": "string"
          }
        }
      },
```

このとき JSON のキーや値をクリップボードにコピーする便利なキーバインドがあります。

`yy` とタイプするとインデントされた JSON がコピーされます。

例:

```json
{
  "type": "object",
  "required": [
    "id",
    "name"
  ],
  "properties": {
    "id": {
      "type": "integer",
      "format": "int64"
    },
    "name": {
      "type": "string"
    },
    "tag": {
      "type": "string"
    }
  }
}
```

`yv` とタイプするとインデントなしの JSON がコピーされます。

例:

```json
{ "type": "object", "required": ["id", "name"], "properties": { "id": { "type": "integer", "format": "int64" }, "name": { "type": "string" }, "tag": { "type": "string" } } }
```

`yk` とタイプすると JSON のキーがコピーされます。

例:

```txt
Pet
```

`yp` とタイプするとその要素へのパスがコピーされます。これは一般的なプログラミング言語でのフィールドへのアクセスで使えるような記法です。

```txt
.components.schemas.Pet
```

`yb` とタイプするとその要素へのパスがコピーされます。これは一般的なプログラミング言語でのマップ、ハッシュ、連想配列、オブジェクトなどへのアクセスで使えるような記法です。

```txt
["components"]["schemas"]["Pet"]
```

`yq` とタイプするとその要素へのパスがコピーされます。これは `jq` で使える記法です。

```txt
.components.schemas.Pet
```

この例では `yp` と `yq` の結果が全く同じですが、配列内にある要素をフォーカスした状態でタイプした際の挙動が異なります。
`yp` の場合は `.[1]` など添字が明示されますが、 `yq` の場合は `.[]` となって全要素の取得を意味するフォーマットになります。

## YAML 対応

パイプからデータを流す場合は `--yaml` オプションを使って YAML も同様に処理できます。

パイプではなく `jless` のコマンドライン引数でファイル名を指定するときは拡張子から JSON or YAML が推測されます。

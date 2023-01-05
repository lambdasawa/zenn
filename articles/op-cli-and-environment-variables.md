---
title: "1Password の CLI で環境変数を管理する"
emoji: "㊙️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["1password", "cli", "env"]
published: true
---

## はじめに

現代のアプリケーションは外部サービスのAPIキーなど様々なクレデンシャルを持つことが多いです。
これらを開発者間で安全に共有するには [sops](https://github.com/mozilla/sops)、 [doppler](https://docs.doppler.com/docs/cli)、 [git-crypt](https://github.com/AGWA/git-crypt) などのツールが使えます。

また、開発時はこれらのクレデンシャルを [direnv](https://github.com/direnv/direnv) などを使って環境変数に設定することも多いのではないでしょうか。

しかし、これらはどれも追加のツールをインストールする必要があります。
もし1Passwordを使っているチームに属しているなら1Passwordでクレデンシャルを管理して、それを環境変数にセットできると新たなツールを導入する必要がなくて楽です。

この記事ではそれを実現する手順を紹介します。

## CLIの設定

まず1PasswordのCLIをインストールします。これで `op` コマンドが使えるようになります。

```sh
brew install --cask 1password/tap/1password-cli
```

`op signin` コマンドでサインインをすると、セッションを `export` するコマンドが出力されるのでそれを `eval` します。
これが成功すると1Passwordの認証が済んだ状態になって、各アイテムにアクセスできます。

```sh
eval $(op signin)
```

## 変数の設定

`foo-vault` という保管庫に `foo-item` というタイトルのセキュアノートを作る際は以下のコマンドを実行します。

```sh
op item create --vault foo-vault --title foo-item --category 'Secure Note'
```

例として環境変数を2つ追加します。
ここでは `foo-section` というセクションに値を追加することにします。セクションは省略可能ですが、省略した場合は自動でランダムな名前が付きます。

```sh
op item edit --vault foo-vault foo-item foo-section.FOO=bar
op item edit --vault foo-vault foo-item foo-section.FIZZ=buzz
```

ここでは `op item edit` コマンドを使用しましたが、もちろんGUI上から設定しても良いです。

1Password上では以下の画像のように表示されます。`FIZZ` だけ `Reveal` しています。

![secure note](https://storage.googleapis.com/zenn-user-upload/787756aaa8fa-20230106.png)

## 変数の参照

例えば `foo-vault` という保管庫に `foo-item` という名前のアイテムがある場合は `op item get --format json --vault foo-vault foo-item` というコマンドで以下のような JSON を取得できます。

```json
{
  "id": "XXXXXXXXXXXXXXXX",
  "title": "foo-item",
  "version": 4,
  "vault": {
    "id": "XXXXXXXXXXXXXXXX",
    "name": "foo-vault"
  },
  "category": "SECURE_NOTE",
  "last_edited_by": "XXXXXXXXXXXXXXXX",
  "created_at": "2023-01-05T23:03:45Z",
  "updated_at": "2023-01-05T23:05:14Z",
  "sections": [
    {
      "id": "XXXXXXXXXXXXXXXX",
      "label": "foo-section"
    }
  ],
  "fields": [
    {
      "id": "notesPlain",
      "type": "STRING",
      "purpose": "NOTES",
      "label": "notesPlain",
      "reference": "op://foo-vault/foo-item/notesPlain"
    },
    {
      "id": "XXXXXXXXXXXXXXXX",
      "section": {
        "id": "XXXXXXXXXXXXXXXX",
        "label": "foo-section"
      },
      "type": "CONCEALED",
      "label": "FOO",
      "value": "bar",
      "reference": "op://foo-vault/foo-item/foo-section/FOO"
    },
    {
      "id": "XXXXXXXXXXXXXXXX",
      "section": {
        "id": "XXXXXXXXXXXXXXXX",
        "label": "foo-section"
      },
      "type": "CONCEALED",
      "label": "FIZZ",
      "value": "buzz",
      "reference": "op://foo-vault/foo-item/foo-section/FIZZ"
    }
  ]
}
```

`jq` コマンドでいい感じにクエリしたりフォーマットすることによって `.env` ファイルを作ったり、 GitHub Actions にこの設定を同期したりするすることが可能です。

`.env` ファイルを作る例:

```sh
op item get --format json --vault foo-vault foo-item | jq -r '.fields[] | select(.value) | (.label) + "=" + (.value)'
```

GitHub Actions にこの設定を同期する例:

```sh
eval $(op item get --format json --vault foo-vault foo-item | jq -r '.fields[] | select(.value) | "gh secret set " + (.label) + " -b \"" + (.value) + "\""')
```

## secret reference

`op` コマンドでは secret reference と呼ばれる以下のような構文が使えます。
先程の `op item get` コマンドの出力に含まれる `reference` というフィールドがそれです。

```txt
op://vault-name/item-name/section-name]field-name
```

この secret reference を `op read` コマンドに渡すことによって、値を1つだけ取り出すことができます。

```sh
$ op read op://foo-vault/foo-item/foo-section/FOO
bar
```

## .env ファイルを生成するアプローチ

`op read` コマンドを活用すると色々なことができますが、ファイルの一部分に 1Password 上の値を埋め込みたいだけであれば `op inject` コマンドを使うと便利です。
`op read` コマンドと同様に secret reference が使用できます。

```sh
$ cat .gitignore
.env

$ cat .env.template
HOGE=op://lambdasawa-sandbox/op-cli/default/HOGE

$ op inject -i .env.template -o .env

$ cat .env
HOGE=hogehogehoge
```

`op inject` コマンドは `.env` のフォーマットに限らず、 YAML や JSON などのファイルに対しても使用できます。

## 環境変数設定済みのシェルを立ち上げるアプローチ

`op inject` コマンドでファイルに値を埋め込むことができるので、 `op run` コマンドと `direnv` などの仕組みを使って 1Password の値を環境変数に埋め込むことができます。
しかし `op run` コマンドを使うと `direnv` などに依存せずに環境変数を設定することができます。

```sh
$ cat .env.template
FOO=op://foo-vault/foo-item/foo-section/FOO

# このシェルでは環境変数が未設定なので何も出力されない
$ echo $FOO

# このコマンドで環境変数が設定された状態で bash が起動する
$ op run --env-file=.env.template --no-masking bash

# 環境変数が取れることが確認できる
$ echo $FOO
bar

$ exit
exit

# exit したら未設定の状態の戻るので何も出力されない
$ echo $FOO
```

## 参考情報

- <https://developer.1password.com/docs/cli/secret-references>
- <https://developer.1password.com/docs/cli/secrets-config-files>

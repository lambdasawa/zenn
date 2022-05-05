---
title: "1Password の CLI で環境変数を管理する"
emoji: "㊙️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["1password", "cli", "env"]
published: false
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

`foo` というアプリを作ることを想定して `foo-app-environment-variables` というセキュアノートを作ります。

```sh
op item create --title foo-app-environment-variables --category 'Secure Note'
```

例として環境変数を2つ追加します。

```sh
op item edit foo-app-environment-variables FOO=bar
op item edit foo-app-environment-variables FIZZ=buzz
```

ここでは `op item edit` コマンドを使用しましたが、もちろんGUI上から設定しても良いです。

1Password上では以下の画像のように表示されます。`FIZZ` だけ `Reveal` しています。

![secure-note](https://storage.googleapis.com/zenn-user-upload/90274bc60d25-20220505.png)

## 変数の参照

`op item get --format json` でセキュアノートの各フィールドをJSON形式で出力できます。
このJSONを `jq` を使って `export FOO=bar` の形式に整えて `eval` します。

```sh
eval $(op item get --format json foo-app-environment-variables | jq -r '.fields[] | select(.value) | "export " + (.label) + "=\"" + (.value) + "\""')
```

そうすると現在のシェルに環境変数が設定された状態になります。

```sh
$ echo $FOO
bar
```

## ユーティリティの用意

ここまでセットアップが完了したら例えば `env.sh` のような名前でシェルスクリプトを用意して、長いコマンドを入力しなくても環境変数を設定できるようにしておくと便利でしょう。

```sh
$ echo $FOO
(何も表示されない)

$ cat env.sh
eval $(op signin)
eval $(op item get --format json foo-app-environment-variables | jq -r '.fields[] | select(.value) | "export " + (.label) + "=\"" + (.value) + "\""')

$ source env.sh

$ echo $FOO
bar
```

こうすることで最初に誰かがこの設定した後、他の人は `source env.sh` を実行するだけで1Password上の環境変数をシェルに反映できます。

## (応用例) GitHub Actionsと連携する

`gh` コマンドを併用すればGitHub Actionsに同様のシークレットを設定するのも簡単です。

```sh
eval $(op item get --format json foo-app-environment-variables | jq -r '.fields[] | select(.value) | "gh secret set " + (.label) + " -b \"" + (.value) + "\""')
```

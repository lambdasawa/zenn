---
title: "1Password の CLI で環境変数を管理する"
emoji: "㊙️"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["1password", "cli", "env"]
published: false
---

## はじめに

現代のアプリケーションは外部サービスのAPIキーなど様々なクレデンシャルを持つことが多いです。
これらを開発者間で共有するには [sops](https://github.com/mozilla/sops)、 [doppler](https://docs.doppler.com/docs/cli)、 [git-crypt](https://github.com/AGWA/git-crypt) などのツールが使えます。

また、開発時はこれらのクレデンシャルを [direnv](https://github.com/direnv/direnv) などを使って環境変数に設定することも多いのではないでしょうか。

しかし、これらはどれも追加のツールをインストールする必要があります。
もし1Passwordを使っているチームに属しているなら1Passwordでクレデンシャルを管理して、それを環境変数にセットできると便利そうです。

この記事ではそれを実現する手順を紹介します。

## 設定方法

まず1PasswordのCLIをインストールします。これで `op` コマンドが使えるようになります。

```sh
brew install --cask 1password/tap/1password-cli
```

`op signin` コマンドでサインインをすると、セッションを `export` するコマンドが出力されるのでそれを `eval` します。
これが成功すると1Passwordの認証が済んだ状態になって、各アイテムにアクセスできます。

```sh
eval $(op signin)
```

次に `foo` というアプリを作ることを想定して `foo-app-environment-variables` というセキュアノートを作ります。

```sh
op item create --title foo-app-environment-variables --category 'Secure Note'
```

例として環境変数を2つ追加します。

```sh
op item edit foo-app-environment-variables FOO=bar
op item edit foo-app-environment-variables FIZZ=buzz
```

1Password上では以下の画像のように表示されます。
ここでは `FIZZ` だけ `Reveal` しています。

![secure-note](https://storage.googleapis.com/zenn-user-upload/90274bc60d25-20220505.png)

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

`gh` コマンドを併用すればGitHub Actionsに同様のシークレットを設定するのも簡単です。

```sh
eval $(op item get --format json foo-app-environment-variables | jq -r '.fields[] | select(.value) | "gh secret set " + (.label) + " -b \"" + (.value) + "\""')
```

## 利用方法

チーム内の誰かが上記の設定したら、他のメンバーは `op signin` と `op item get` を行うだけでクレデンシャルを環境変数に反映できます。

```sh
eval $(op signin)
eval $(op item get --format json foo-app-environment-variables | jq -r '.fields[] | select(.value) | "export " + (.label) + "=\"" + (.value) + "\""')
```

## おわりに

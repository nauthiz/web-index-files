# Web Index Files CLI

## 概要

Webサーバが返すインデックスページを読み込み、含まれているファイルを表示・ダウンロードするツール。
現状はNginxのみ対応。

## 使い方

指定したURLに含まれるファイル、ディレクトリを一覧表示。

```
$ web-index-files list https://path/to/index
```

指定したURLに含まれるファイル、ディレクトリを再帰的に一覧表示。

```
$ web-index-files list https://path/to/index -r
```

指定したURLに含まれるファイル、ディレクトリをoutputディレクトリに保存。

```
$ web-index-files dl https://path/to/index -o output
```

Basic認証が必要な場合は `-a` でユーザとパスワードを指定。

```
$ web-index-files list https://path/to/index -a "USER:PASSWORD"
```

## ビルド方法

```
$ go build
```

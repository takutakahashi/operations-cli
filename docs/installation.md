# インストール方法

Operations CLIには複数のインストール方法があります。

## ワンライナーインストール（推奨）

最新バージョンを簡単にインストールするには、以下のコマンドを実行してください：

```bash
curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh
```

特定のバージョンをインストールする場合は、以下のように`-v`オプションを使用します：

```bash
curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh -s -- -v 1.0.0
```

カスタムディレクトリにインストールする場合は、`-d`オプションを使用します：

```bash
curl -fsSL https://takutakahashi.github.io/operation-mcp/install.sh | sh -s -- -d ~/bin
```

### インストールスクリプトのオプション

インストールスクリプトは以下のオプションをサポートしています：

| オプション | 説明 |
|------------|------|
| `-v, --version VERSION` | インストールするバージョンを指定します（デフォルト: 最新） |
| `-d, --dir DIRECTORY` | インストール先ディレクトリを指定します（デフォルト: /usr/local/bin） |
| `-f, --force` | 確認プロンプトをスキップします |
| `--dry-run` | 変更を加えずに何が行われるかを表示します |
| `-h, --help` | ヘルプメッセージを表示します |

## ソースからのビルド

Goの開発環境がある場合は、リポジトリからソースコードをクローンしてビルドすることもできます：

```bash
# リポジトリをクローン
git clone https://github.com/takutakahashi/operation-mcp.git
cd operation-mcp

# バイナリをビルド
make build

# インストール（オプション）
make install
```

## 既存インストールのアップグレード

すでにインストールされているOperations CLIをアップグレードする場合は、`upgrade`コマンドを使用できます：

```bash
# 最新バージョンにアップグレード
operations upgrade

# 特定のバージョンにアップグレード
operations upgrade --version v1.0.0

# 利用可能なバージョンを確認（アップグレードなし）
operations upgrade --dry-run

# 確認プロンプトなしでアップグレード
operations upgrade --force
```

詳細なアップグレードオプションについては、`operations upgrade --help`を参照してください。

## インストール後の確認

インストールが成功したかを確認するには、以下のコマンドを実行します：

```bash
operations --version
```

バージョン情報が表示されれば、インストールは正常に完了しています。
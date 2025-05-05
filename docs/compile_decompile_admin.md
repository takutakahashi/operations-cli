# config compile / decompile 管理者向けドキュメント

## 概要

`config compile` および `config decompile` は、ディレクトリ構造とYAML形式の設定ファイル（config）を相互変換するためのコマンドです。

- `compile`: ディレクトリ（metadata.yaml, スクリプト群）→ 1つのYAML設定ファイルにまとめる
- `decompile`: YAML設定ファイル → ディレクトリ・ファイル群に展開する

これにより、設定のバージョン管理やCI/CD、運用現場での柔軟な管理が可能になります。

---

## ディレクトリ構造と metadata.yaml の書き方

### 推奨ディレクトリ構成例

```
<root-dir>/
├── metadata.yaml           # ルートのアクション・ツール一覧
├── tools/
│   ├── <tool名>/
│   │   ├── metadata.yaml   # ツールのパラメータ・スクリプト・サブツール定義
│   │   ├── main.sh        # scriptで指定した場合のスクリプト本体
│   │   ├── beforeExec/    # beforeExecスクリプト群
│   │   ├── afterExec/     # afterExecスクリプト群
│   │   └── <subtool名>/   # サブツールも同様にディレクトリで管理
│   │       ├── metadata.yaml
│   │       └── main.sh
│   └── ...
└── ...
```

### ルート metadata.yaml の例

```yaml
actions:
  - danger_level: low
    type: confirm
    message: "This is a low-risk operation. Proceed?"
  - danger_level: medium
    type: confirm
    message: "This is a medium-risk operation. Are you sure you want to proceed?"
  - danger_level: high
    type: confirm
    message: "This is a high-risk operation. Please confirm carefully before proceeding."
tools:
  - path: ./tools/bash
  - path: ./tools/lifecycle
```

### ツール/サブツール metadata.yaml の例

```yaml
params:
  param1:
    description: 説明
    type: string
    required: true
script: main.sh
beforeExec:
  - path: beforeExec/00-echo.sh
afterExec:
  - path: afterExec/00-echo.sh
# サブツールがある場合
tools:
  - path: subtool1
  - path: subtool2
```

- `params`: パラメータ定義（name/type/description/required など）
- `script`: 実行スクリプトファイル名（同ディレクトリ内のファイル）
- `beforeExec`/`afterExec`: 実行前後のスクリプト（`path`でファイル指定）
- `tools`: サブツールのディレクトリパス

### beforeExec/afterExec の例

`beforeExec/00-echo.sh`:
```sh
#!/bin/bash
echo "Before exec"
```

---

## 使い方

### ディレクトリからYAMLを生成（compile）

```sh
operations config compile -d <ディレクトリ> -o <出力YAMLファイル>
```

例:

```sh
operations config compile -d misc/generate -o config.yaml
```

- `-d`/`--dir`: ベースとなるディレクトリ（metadata.yaml等が含まれる）
- `-o`/`--output`: 出力先YAMLファイル（省略時は標準出力）

### YAMLからディレクトリを生成（decompile）

```sh
operations config decompile -f <YAMLファイル> -d <出力ディレクトリ>
```

例:

```sh
operations config decompile -f config.yaml -d tmp/dir
```

- `-f`/`--file`: 入力YAMLファイル
- `-d`/`--dir`: 出力先ディレクトリ

---

## 注意点・ベストプラクティス

- **スクリプトやbeforeExec/afterExecはファイル内容がインライン展開されます。**
- compile/decompileを往復しても意味が変わらないことをe2eテストで保証しています。
- ディレクトリ構造の例は `misc/generate` を参照してください。
- YAMLの差分比較はインデントや順序の違いを許容する場合があります。CIでは内容一致を重視してください。
- `actions`（danger_levelごとのconfirm等）も正しく展開・復元されます。

---

## 活用例

- **Git管理**: ディレクトリ構造で細かくレビューしやすく、リリース時にYAMLにまとめてデプロイ
- **CI/CD**: PR時に `compile` でYAML生成→テスト→本番反映
- **現場運用**: 設定の一括配布や、部分的なスクリプト修正もディレクトリ単位で容易

---

## トラブルシューティング

- YAMLパースエラーが出る場合は、`|4` などのインデント記法が `|-` に正規化されているか確認してください（現行実装では自動で置換されます）。
- スクリプトの実行権限や改行コードにも注意してください。
- e2eテスト（`misc/run_compile_decompile_e2e.sh`）で往復変換・execまで自動検証できます。

---

## 参考
- `misc/generate` ... サンプルディレクトリ構造
- `misc/run_compile_decompile_e2e.sh` ... e2eテストスクリプト
- `.github/workflows/e2e-test.yml` ... CIでの自動テスト例 
# Operations CLI ツール仕様書

## 概要

Operations CLI は、設定ファイルから動的にツール群を生成し、実行できるようにするコマンドラインツールです。

## 基本仕様

- 言語: Go 1.19
- コマンド名: `operations`
- 設定ファイル: YAML形式
- ツール名の命名規則: すべての単語をアンダースコア（_）でつなげた形式
  - 例: `kubectl_get_pod`, `kubectl_describe_pod`

## 危険度管理仕様

### 背景と目的

このツールは、CI/CDパイプラインや自動化スクリプトなど、機械的な実行環境での使用を想定しています。そのため、以下の理由から危険度管理が重要となります：

1. **自動実行時の安全性確保**
   - 機械的な実行環境では、人間による確認ができない
   - 誤った操作による重大な影響を防ぐ必要がある

2. **実行環境の保護**
   - 本番環境や重要なシステムに対する操作を制御
   - 意図しない操作によるシステムダウンを防止

3. **操作の透明性確保**
   - 実行される操作の危険度を明示
   - 自動化環境での操作ログの追跡を容易に

### 設定構造

```yaml
actions:
  - danger_level: <危険度>
    type: <アクションタイプ>
    message: <確認メッセージ>
    timeout: <タイムアウト秒数>
```

### 設定項目の説明

1. **アクション設定 (actions)**
   - 危険度レベルごとのアクション設定
   - 以下の属性を持つ：
     - danger_level: 危険度レベル
     - type: アクションタイプ（confirm, timeout, force）
     - message: 確認メッセージ
     - timeout: タイムアウト秒数

### 機能要件

1. **セキュリティ機能**
   - 危険度レベルに基づく確認プロンプト
   - 除外対象のチェック
   - 危険度に応じたアクション実行
     - confirm: ユーザー確認を要求
     - timeout: 指定秒数後に自動実行
     - force: 強制実行（警告のみ表示）

## 設定インポート機能

### 設定構造

```yaml
imports:
  - <インポートするファイルのパス>
  - <別のファイルパス>
```

### 設定項目の説明

1. **インポート (imports)**
   - インポートする設定ファイルパスのリスト
   - 相対パスは、メインの設定ファイルからの相対パスとして解釈
   - 絶対パスも指定可能
   - 複数のファイルをインポート可能

### 機能要件

1. **設定ファイルの結合**
   - インポートした設定ファイルの内容をメインの設定と結合
   - アクションリストはすべてのファイルから結合
   - ツールリストもすべてのファイルから結合
   - 同名のツールが複数のファイルで定義されている場合、メインファイルでの定義が優先
   - SSHなどの単一の設定は、メインファイルでの定義が優先

2. **循環インポートの検出**
   - 循環参照を検出して、無限ループを防止
   - 循環参照が検出された場合はエラーメッセージを表示

### 使用例

```yaml
# メインの設定ファイル
actions:
  - danger_level: high
    type: confirm
    message: "高危険度の操作です。続行しますか？"

imports:
  - team/database-tools.yaml
  - team/network-tools.yaml

tools:
  - name: kubectl
    command:
      - kubectl
    # ... (他の設定)
```

```yaml
# team/database-tools.yaml
actions:
  - danger_level: medium
    type: timeout
    message: "データベース操作を開始します。"
    timeout: 10

tools:
  - name: db-tool
    command:
      - db-tool
    # ... (他の設定)
```

## ツール仕様

### 設定構造

```yaml
tools:
  - name: <ツール名>
    command: [<コマンド>, ...]
    beforeExec: <実行前に実行するスクリプト>
    afterExec: <実行後に実行するスクリプト>
    params:
      <パラメータ名>:
        description: <パラメータの説明>
        type: <パラメータの型>
        required: <必須かどうか>
        validate:
          - danger_level: <危険度>
            exclude: [<除外対象>, ...]
    subtools:
      - name: <サブツール名>
        args: [<引数>, ...]
        beforeExec: <実行前に実行するスクリプト>
        afterExec: <実行後に実行するスクリプト>
        params:
          <パラメータ名>:
            description: <パラメータの説明>
            type: <パラメータの型>
            required: <必須かどうか>
        danger_level: <危険度>
        subtools:
          - name: <子サブツール名>
            args: [<引数>, ...]
            beforeExec: <実行前に実行するスクリプト>
            afterExec: <実行後に実行するスクリプト>
            params:
              <パラメータ名>:
                description: <パラメータの説明>
                type: <パラメータの型>
                required: <必須かどうか>
            danger_level: <危険度>
```

### 設定項目の説明

1. **ツール (tools)**
   - 最上位の設定項目
   - 複数のツールを定義可能

2. **ツール名 (name)**
   - ツールを識別するための名前
   - コマンドラインで使用される

3. **コマンド (command)**
   - 実行するコマンドの配列
   - 最初の要素が実行ファイル名、以降がデフォルト引数

4. **ライフサイクルフック (beforeExec, afterExec)**
   - beforeExec: ツールの実行前に実行されるスクリプト
     - シェルスクリプト形式で記述
     - パラメータをテンプレート変数として使用可能 (例: {{.param}})
     - 親ツールのbeforeExecが最初に実行され、その後子ツールのbeforeExecが実行される
   - afterExec: ツールの実行後に実行されるスクリプト
     - シェルスクリプト形式で記述
     - パラメータをテンプレート変数として使用可能
     - 子ツールのafterExecが最初に実行され、その後親ツールのafterExecが実行される
   - 実行順序: parent.beforeExec → child.beforeExec → main script → child.afterExec → parent.afterExec

5. **パラメータ (params)**
   - ツール実行時に必要なパラメータの定義
   - 各パラメータは以下の属性を持つ：
     - description: パラメータの説明
     - type: パラメータの型（string, number, boolean など）
     - required: 必須かどうか
     - validate: バリデーションルール
   - パラメータの継承
     - 親ツールのパラメータは、すべての子サブツールに自動的に継承される
     - 子サブツールで同名のパラメータを定義した場合、子の定義が優先される
     - 継承されたパラメータは、コマンドラインで指定可能

6. **サブツール (subtools)**
   - ツールのサブコマンド
   - 各サブツールは以下の属性を持つ：
     - name: サブツール名
     - args: 実行時の引数
     - params: サブツール固有のパラメータ
     - danger_level: 危険度レベル
     - subtools: 子サブツールの定義（オプション）
       - 子サブツールも同様の構造を持つ
       - 再帰的に定義可能
   - パラメータの継承
     - 親ツールのパラメータをすべて継承
     - 継承されたパラメータは、コマンドラインで指定可能

### 機能要件

1. **設定ファイルの読み込み**
   - YAML形式の設定ファイルを読み込む
   - 設定のバリデーションを行う

2. **コマンド生成**
   - 設定ファイルに基づいて動的にコマンドを生成
   - パラメータの置換処理を行う

3. **コマンド実行**
   - 生成されたコマンドを実行
   - 実行結果の表示

## 使用例

```bash
# 設定ファイルに基づいてツールを実行
operations kubectl_get_pod --namespace my-namespace

# サブツールの実行
operations kubectl_describe_pod --namespace my-namespace --pod my-pod

# 子サブツールの実行
operations kubectl_logs_container --namespace my-namespace --pod my-pod --container my-container

# 親ツールのパラメータ（namespace）を子サブツールで使用
operations kubectl_logs_container --namespace my-namespace --pod my-pod --container my-container

# 親ツールと子サブツールのパラメータを組み合わせて使用
operations kubectl_exec_command --namespace my-namespace --pod my-pod --container my-container --command "ls -la"
```

## 制約事項

1. Go 1.19 の機能のみを使用
2. 設定ファイルは YAML 形式のみ対応
3. コマンド実行はシェル経由で行う 
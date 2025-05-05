#!/bin/bash
set -euo pipefail

# 作業ディレクトリ
E2E_DIR="./tmp/e2e"
GENSRC="misc/generate"
OUT1="$E2E_DIR/out1.yaml"
TMPDIR="$E2E_DIR/dir"
OUT2="$E2E_DIR/out2.yaml"

mkdir -p "$E2E_DIR"
mkdir -p "$TMPDIR"

operations() {
  go run main.go "$@"
}

echo "== compile: ディレクトリ -> YAML =="
operations config compile -d "$GENSRC" -o "$OUT1"

echo "== decompile: YAML -> ディレクトリ =="
operations config decompile -f "$OUT1" -d "$TMPDIR"

echo "== 再compile: ディレクトリ -> YAML =="
operations config compile -d "$TMPDIR" -o "$OUT2"

# 期待される出力（misc/generate/output.yaml）との比較も行う
if [ -f "$GENSRC/output.yaml" ]; then
  echo "== 期待される出力とのdiff =="
  if diff -u "$OUT1" "$GENSRC/output.yaml"; then
    echo "OK: 期待される出力と一致"
  else
    echo "NG: 期待される出力と一致しません"
    exit 1
  fi
fi

echo "== 生成YAMLでのexecテスト =="
OPERATIONS_BIN="go run main.go -c $OUT1"

# bash/loop
if ! $OPERATIONS_BIN exec bash_loop --set iterations=2 | grep -q "Loop completed"; then
  echo "NG: bash_loop のexecに失敗"
  exit 1
fi
# bash/variables
if ! $OPERATIONS_BIN exec bash_variables --set name="E2E" --set count=3 | grep -q "Hello, E2E!"; then
  echo "NG: bash_variables のexecに失敗"
  exit 1
fi
# bash/conditional
if ! echo "y" | $OPERATIONS_BIN exec bash_conditional --set value=5 | grep -q "less than or equal"; then
  echo "NG: bash_conditional のexecに失敗"
  exit 1
fi
# lifecycle/test
expected_lifecycle_test_output=$'Root before_exec\nSubtool before_exec\nMain script execution\nSubtool after_exec\nRoot after_exec'
actual_lifecycle_test_output=$($OPERATIONS_BIN exec lifecycle_test)
if [[ "$actual_lifecycle_test_output" != *"Main script execution"* ]]; then
  echo "NG: lifecycle_test のexecに失敗 (Main script executionが出力されていません)"
  exit 1
fi
# 順序も厳密に比較
if ! diff <(echo "$expected_lifecycle_test_output") <(echo "$actual_lifecycle_test_output") >/dev/null; then
  echo "NG: lifecycle_test の出力順序が期待と異なります"
  echo "期待される出力:"
  echo "$expected_lifecycle_test_output"
  echo "実際の出力:"
  echo "$actual_lifecycle_test_output"
  exit 1
fi
# lifecycle/with-params
if ! $OPERATIONS_BIN exec lifecycle_with-params --set param="foo" | grep -q "Main script with param: foo"; then
  echo "NG: lifecycle_with-params のexecに失敗"
  exit 1
fi

echo "すべてのe2eテストが成功しました" 

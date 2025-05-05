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

echo "== YAML同士のdiff =="
if diff -u "$OUT1" "$OUT2"; then
  echo "OK: compile/decompileの往復で差分なし"
else
  echo "NG: 差分があります"
  exit 1
fi

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

echo "すべてのe2eテストが成功しました" 

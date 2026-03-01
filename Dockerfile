# ビルドステージ
FROM golang:1.24-bullseye AS builder

WORKDIR /app

# ソースコードをコピー
COPY . .

# バイナリのビルド
RUN go build -o operations .

# 実行ステージ
# Node.js ベースイメージを使用して supergateway (remote MCP) をサポート
FROM node:20-slim

WORKDIR /app

# supergateway をグローバルインストール
RUN npm install -g supergateway

# curl をインストール
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

# ビルドステージからバイナリをコピー
COPY --from=builder /app/operations /usr/local/bin/operations

# エントリーポイントスクリプトをコピー
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# remote MCP 用のポートを公開 (MCP_MODE=remote 時に使用)
EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# ビルドステージ
FROM golang:1.24-bullseye AS builder

WORKDIR /app

# ソースコードをコピー
COPY . .

# バイナリのビルド
RUN go build -o operations .

# 実行ステージ
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# ビルドステージからバイナリをコピー
COPY --from=builder /app/operations /app/operations
USER nonroot

ENTRYPOINT ["/app/operations"]

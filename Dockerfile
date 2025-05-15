# ビルドステージ
FROM golang:1.24-bullseye AS builder

WORKDIR /app

# ソースコードをコピー
COPY . .

# バイナリのビルド
RUN CGO_ENABLED=0 GOOS=linux go build -o operations ./cmd/operations

# 実行ステージ
FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# ビルドステージからバイナリをコピー
COPY --from=builder /app/operations /app/operations

# 実行権限の確保
USER root
RUN chmod +x /app/operations
USER nonroot

ENTRYPOINT ["/app/operations"]
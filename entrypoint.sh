#!/bin/sh
set -e

# MCP_MODE=remote の場合、supergateway 経由でリモート MCP サーバーとして起動する
# それ以外の場合は operations バイナリを直接実行する (stdio モード)
if [ "${MCP_MODE}" = "remote" ]; then
    PORT=${PORT:-8000}
    SUPERGATEWAY_ARGS="${SUPERGATEWAY_ARGS:-}"
    exec supergateway \
        --stdio "operations $*" \
        --port "${PORT}" \
        ${SUPERGATEWAY_ARGS}
else
    exec operations "$@"
fi

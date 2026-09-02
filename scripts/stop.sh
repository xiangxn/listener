#!/bin/bash
# 一键停止: listener(模拟盘) + MongoDB 容器
# MongoDB 数据持久化在 data/mongodb, 停止后不丢, 下次 scripts/start.sh 自动拉起
# 仅停止本脚本启动的进程, 不影响其他项目

set -u

# 脚本位于 scripts/, 统一切到仓库根目录操作(logs 在根目录)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

LOG_DIR="logs"
PID_FILE="${LOG_DIR}/listener.pid"
CONTAINER="listener-mongodb"

# ---------- 1. 停止 listener ----------
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
        echo "停止 listener (pid $PID)..."
        kill -TERM "$PID"   # bot 捕获 SIGTERM 优雅退出(关闭 WS 与 DB 连接)
        stopped=""
        for _ in $(seq 1 10); do
            if ! kill -0 "$PID" 2>/dev/null; then stopped=1; break; fi
            sleep 1
        done
        if [ -z "$stopped" ]; then
            echo "10s 内未退出, 强制结束"
            kill -9 "$PID" 2>/dev/null
        fi
    else
        echo "listener 未在运行(pid 文件已过期)"
    fi
    rm -f "$PID_FILE"
else
    # 没有 pid 文件时兜底: 清理残留的 listener arb 进程
    PIDS=$(pgrep -f "[.]/listener arb" || true)
    if [ -n "$PIDS" ]; then
        echo "清理残留进程: $PIDS"
        kill -TERM $PIDS 2>/dev/null
    else
        echo "listener 未在运行"
    fi
fi

# ---------- 2. 停止 MongoDB ----------
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${CONTAINER}$"; then
    echo "停止 MongoDB 容器 ${CONTAINER}..."
    docker stop ${CONTAINER} >/dev/null 2>&1 && echo "已停止(数据保留在 data/mongodb)"
else
    echo "MongoDB 容器未在运行"
fi

echo "全部已停止 ✓"

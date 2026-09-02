#!/bin/bash
# 本地 Robinhood 模拟盘一键启动
#
# 流程: 环境检查(check.sh, 缺依赖自动安装) → 启动 MongoDB → 编译 listener → 输入钱包密码 → 后台运行
# 停止: scripts/stop.sh
# 日志: tail -f logs/arb.log   (pid 记录在 logs/listener.pid)
#
# 说明:
#   - robinhood.config.yaml 已开 simulation.enable, 监听到套利机会时在本地 anvil fork 上
#     模拟成交(不发真实交易), 需要 foundry 提供 anvil/cast(check.sh 会处理)
#   - 密码仅用于解密 .env 中的 PRIVATE_WIF(模拟盘同样需要, 用于 fork 上的测试账户),
#     不落盘; 也可预先 export LISTENER_PASSWORD=xxx 跳过交互输入

set -u

# 脚本位于 scripts/, 统一切到仓库根目录操作
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

CONFIG="robinhood.config.yaml"
BIN="./listener"
LOG_DIR="logs"
LOG="${LOG_DIR}/arb.log"
PID_FILE="${LOG_DIR}/listener.pid"

red='\033[31m'; green='\033[32m'; yellow='\033[33m'; none='\033[0m'
die()  { echo -e "${red}[出错]${none} $*" >&2; exit 1; }
info() { echo -e "${green}[信息]${none} $*"; }

mkdir -p "$LOG_DIR"

# ---------- 0. 环境检查(缺失依赖自动安装) ----------
"$SCRIPT_DIR/check.sh" "$CONFIG" || exit 1

# 已在运行则退出
if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    die "listener 已在运行 (pid $(cat "$PID_FILE"))。先执行 scripts/stop.sh 再启动"
fi
[ -f "$PID_FILE" ] && rm -f "$PID_FILE" # 清理过期 pid 文件

# ---------- 1. MongoDB ----------
info "启动 MongoDB(容器 listener-mongodb, 数据在 data/mongodb)..."
"$SCRIPT_DIR/startdb.sh" >/dev/null 2>&1 || die "startdb.sh 启动 MongoDB 失败"
tries=0
until docker exec listener-mongodb mongosh --quiet --eval 'db.runCommand({ping:1}).ok' 2>/dev/null | grep -q 1; do
    tries=$((tries + 1))
    if [ $tries -ge 120 ]; then
        die "MongoDB 120s 内未就绪(首次启动会拉 mongo:7.0 镜像, 请稍后再试)"
    fi
    [ $tries -eq 5 ] && warn "等待 MongoDB 就绪(首次会拉取镜像, 可能需要几分钟)..."
    sleep 1
done
info "MongoDB 就绪"

# ---------- 2. 编译(源码比二进制新或首次运行才编译) ----------
if [ ! -x "$BIN" ] || find . -name '*.go' -newer "$BIN" 2>/dev/null | grep -q .; then
    info "编译 listener(go build)..."
    go build -o "$BIN" . || die "go build 失败"
fi

# ---------- 3. 输入密码(用于解密 .env 的 PRIVATE_WIF, 不落盘) ----------
PW="${LISTENER_PASSWORD:-}"
if [ -z "$PW" ]; then
    read -s -r -p "$(echo -e "${yellow}[输入]${none} 钱包密码(用于解密 PRIVATE_WIF, 不落盘): ")" PW
    echo
fi
[ -n "$PW" ] || die "密码为空"

# ---------- 4. 启动 ----------
# 把 ~/.foundry/bin 加入 PATH, 供 listener 调 anvil/cast(新装或未 source rc 的情况)
export PATH="$HOME/.foundry/bin:${PATH}"
info "启动 listener arb -c $CONFIG(模拟盘模式)..."
rm -f "$LOG"
export LISTENER_PASSWORD="$PW"
nohup "$BIN" arb -c "$CONFIG" >"$LOG" 2>&1 </dev/null &
PID=$!
echo "$PID" > "$PID_FILE"

sleep 5
if ! kill -0 "$PID" 2>/dev/null; then
    rm -f "$PID_FILE"
    echo "----- 进程已退出, 日志末尾 -----"
    tail -n 40 "$LOG" 2>/dev/null
    die "listener 启动失败, 见上方日志"
fi

info "listener 运行中: pid=$PID (日志 $LOG, 停止: scripts/stop.sh)"
tail -n 3 "$LOG" 2>/dev/null || true

#!/bin/bash
# 本地模拟盘运行环境检查: 缺什么补什么(可独立运行; start.sh 启动时会自动调用)
#
# 检查项:
#   1. 配置文件存在
#   2. docker 可用(跑 MongoDB)
#   3. .env 存在且 PRIVATE_WIF 非空
#   4. foundry(anvil/cast/forge)已安装 — 缺失则自动安装到 ~/.foundry
#
# 用法: ./check.sh [配置文件, 默认 robinhood.config.yaml]
set -u

# 脚本位于 scripts/, 统一切到仓库根目录操作(配置文件/.env/data 都在根目录)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"
CONFIG="${1:-robinhood.config.yaml}"

red='\033[31m'; green='\033[32m'; yellow='\033[33m'; none='\033[0m'
die()  { echo -e "${red}[出错]${none} $*" >&2; exit 1; }
info() { echo -e "${green}[信息]${none} $*"; }
warn() { echo -e "${yellow}[提示]${none} $*"; }

echo "== Robinhood 模拟盘环境检查 =="

# ---------- 1. 配置文件 ----------
[ -f "$CONFIG" ] || die "缺少配置文件 $CONFIG"
info "✓ 配置文件 $CONFIG"

# ---------- 2. docker ----------
command -v docker >/dev/null 2>&1 || die "缺少 docker(MongoDB 需要它)"
info "✓ docker: $(docker --version 2>/dev/null | head -1)"

# ---------- 3. .env / PRIVATE_WIF ----------
[ -f ".env" ] || die "缺少 .env。先执行: cp example.env .env, 再用 ./listener crypto -E <你的私钥> 加密, 把输出密文填入 PRIVATE_WIF(模拟盘也需要, 见 README「本地模拟盘」)"
WIF=$(grep '^PRIVATE_WIF=' .env 2>/dev/null | head -1 | cut -d= -f2-)
[ -n "$WIF" ] || die ".env 中的 PRIVATE_WIF 为空。先用 ./listener crypto -E <私钥> 加密后填入"
info "✓ .env / PRIVATE_WIF 已配置"

# ---------- 4. foundry(anvil/cast/forge) ----------
install_foundry() {
    warn "未检测到 foundry(anvil/cast), 现在自动安装到 ~/.foundry ..."
    command -v curl >/dev/null 2>&1 || die "缺少 curl, 无法下载 foundry"
    if [ ! -x "$HOME/.foundry/bin/foundryup" ]; then
        curl -sS -L https://foundry.paradigm.xyz | bash \
            || die "foundryup 安装失败(需要能访问 github, 可重试或手动安装: curl -L https://foundry.paradigm.xyz | bash)"
    fi
    "$HOME/.foundry/bin/foundryup" || die "foundryup 更新/安装失败(需要网络), 可重试 ./check.sh"
}

if command -v anvil >/dev/null 2>&1 && command -v cast >/dev/null 2>&1; then
    info "✓ foundry: $(command -v anvil)"
elif [ -x "$HOME/.foundry/bin/anvil" ] && [ -x "$HOME/.foundry/bin/cast" ]; then
    # 已装但不在 PATH(例如新开的 shell 没 source rc) — 不需要重装
    info "✓ foundry: $HOME/.foundry/bin(不在 PATH, start.sh 会临时加进去)"
else
    install_foundry
    # 刚装完的目录同样不在当前 shell 的 PATH, 用绝对路径验证
    [ -x "$HOME/.foundry/bin/anvil" ] && [ -x "$HOME/.foundry/bin/cast" ] \
        || die "foundry 安装后仍不可用, 请检查 $HOME/.foundry/bin"
    info "✓ foundry 安装完成: $HOME/.foundry/bin (start.sh 会自动使用)"
fi

echo "== 环境检查通过, 可以启动 =="

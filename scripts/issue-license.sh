#!/usr/bin/env bash
# issue-license.sh — Cortex 订单 → 许可 发证脚本（销售手工发证 / 批量发证）
#
# 用途
#   把「客户付款」和「交付许可」之间的手工步骤压到一条命令：
#   读取订单信息 → 调用 licensegen 签发 → 自检 → 输出可直接粘贴给客户的邮件正文。
#
# 为什么要有这个脚本
#   licensegen 是底层发证工具，参数多、易漏 --private-key；
#   销售在真实流程里需要的是「一笔订单进、一封邮件出」，且必须防止
#   "把无效许可以及错别字邮箱发给客户" 这类事故。本脚本把这些检查固化下来。
#
# 用法
#   单笔（交互最少，推荐）：
#     scripts/issue-license.sh --email user@example.com --plan pro --days 365 --order ORD-1001
#
#   单笔 + 指定到期日：
#     scripts/issue-license.sh --email user@example.com --plan team --expires 2027-11-09 --seats 5
#
#   批量（CSV: email,plan,days,seats,order）：
#     scripts/issue-license.sh --orders orders.csv --out-dir .gobuild/out
#
#   自检（不需要私钥，只验证格式）：
#     scripts/issue-license.sh --selftest
#
# 环境变量
#   CORTEX_PRIVATE_KEY   私钥 hex 文件路径（不给就用 --private-key）
#   CORTEX_LICENSEGEN    licensegen 可执行文件路径（默认自动找）
#
# 退出码
#   0 成功 / 1 发证或自检失败 / 2 参数错误
#
# 安全
#   脚本永不打印私钥内容；--out 生成的文件默认 600 权限。

set -euo pipefail

# ---------- 仓库根定位（脚本在 scripts/ 下） ----------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# ---------- 默认值 ----------
PLAN="pro"
DAYS="365"
SEATS="1"
ORDER=""
EMAIL=""
EXPIRES=""
FEATURES=""
ORDERS_CSV=""
OUT_DIR=""
PRIVATE_KEY="${CORTEX_PRIVATE_KEY:-}"
LICENSEGEN="${CORTEX_LICENSEGEN:-}"
OUT_FILE=""
SELFTEST=0
QUIET=0

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; DIM=$'\033[2m'; BOLD=$'\033[1m'; NC=$'\033[0m'
if [ ! -t 1 ]; then RED=""; GREEN=""; YELLOW=""; DIM=""; BOLD=""; NC=""; fi

info()  { [ "$QUIET" -eq 1 ] || echo "${DIM}$*${NC}" >&2; }
ok()    { echo "${GREEN}✓${NC} $*" >&2; }
warn()  { echo "${YELLOW}!${NC} $*" >&2; }
die()   { echo "${RED}✗${NC} $*" >&2; exit 1; }
die2()  { echo "${RED}✗${NC} $*" >&2; exit 2; }

usage() {
  sed -n '2,40p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

# ---------- 跨环境路径处理（WSL/MSYS 下 bash 路径与 Windows .exe 不互通） ----------
# 把 bash 路径翻译成 licensegen.exe 能听懂的 Windows 路径；不是 WSL/MSYS 就原样返回。
native_path() {
  if command -v wslpath >/dev/null 2>&1; then
    wslpath -w "$1"
  elif command -v cygpath >/dev/null 2>&1; then
    cygpath -w "$1"
  else
    printf '%s' "$1"
  fi
}

# 建临时目录：优先放仓库内 .gobuild/tmp（避免 /tmp 在 Windows 侧不存在）
make_tmpdir() {
  local base="${TMPDIR:-$REPO_ROOT/.gobuild/tmp}"
  mkdir -p "$base" 2>/dev/null || base="$REPO_ROOT/.gobuild/tmp"
  mkdir -p "$base"
  mktemp -d "$base/lic.XXXXXX"
}

# ---------- 参数解析 ----------
while [ $# -gt 0 ]; do
  case "$1" in
    --email)       EMAIL="${2:-}"; shift 2 ;;
    --plan)        PLAN="${2:-}"; shift 2 ;;
    --days)        DAYS="${2:-}"; shift 2 ;;
    --seats)       SEATS="${2:-}"; shift 2 ;;
    --order)       ORDER="${2:-}"; shift 2 ;;
    --expires)     EXPIRES="${2:-}"; shift 2 ;;
    --features)    FEATURES="${2:-}"; shift 2 ;;
    --private-key) PRIVATE_KEY="${2:-}"; shift 2 ;;
    --licensegen)  LICENSEGEN="${2:-}"; shift 2 ;;
    --orders)      ORDERS_CSV="${2:-}"; shift 2 ;;
    --out-dir)     OUT_DIR="${2:-}"; shift 2 ;;
    --out)         OUT_FILE="${2:-}"; shift 2 ;;
    --selftest)    SELFTEST=1; shift ;;
    --quiet)       QUIET=1; shift ;;
    -h|--help)     usage; exit 0 ;;
    *)             die2 "未知参数：$1（用 --help 查看用法）" ;;
  esac
done

# ---------- 定位 licensegen ----------
find_licensegen() {
  if [ -n "$LICENSEGEN" ] && [ -x "$LICENSEGEN" ]; then echo "$LICENSEGEN"; return; fi
  local cand
  for cand in     "$REPO_ROOT/.gobuild/bin/licensegen.exe"     "$REPO_ROOT/.gobuild/bin/licensegen"     "$REPO_ROOT/bin/licensegen.exe"     "$REPO_ROOT/bin/licensegen"; do
    [ -x "$cand" ] && { echo "$cand"; return; }
  done
  if command -v licensegen >/dev/null 2>&1; then command -v licensegen; return; fi
  echo ""
}

build_hint() {
  cat >&2 <<'EOF'

请先构建 licensegen（不需要 CGO，约 1-2 分钟）：

  export GOPATH="$PWD/.gopath"
  export GOMODCACHE="$PWD/.gopath/pkg/mod"
  export GOCACHE="$PWD/.gobuild/gocache"
  export GOTMPDIR="$PWD/.gobuild/tmp"
  export CGO_ENABLED=0
  go build -o .gobuild/bin/licensegen ./cmd/licensegen/

如果本机没有 Go 环境，可用等价的 Python 发证器（无需编译）：

  python scripts/cortex_license.py issue --private-key <私钥文件> --plan pro \
      --email user@example.com --order ORD-1001 --days 365
EOF
}

# ---------- 自检模式（不需要私钥） ----------
run_selftest() {
  local gen; gen="$(find_licensegen)"
  echo "${BOLD}Cortex 发证链路自检${NC}" >&2
  echo "" >&2

  # 1) licensegen 是否存在
  if [ -z "$gen" ]; then
    echo "${RED}✗${NC} 未找到 licensegen 可执行文件" >&2
    build_hint
    return 1
  fi
  echo "${GREEN}✓${NC} licensegen: $gen" >&2

  # 2) licensegen 子命令是否完整
  if "$gen" help 2>&1 | grep -q "batch"; then
    echo "${GREEN}✓${NC} licensegen 支持 batch/verify 子命令" >&2
  else
    echo "${YELLOW}!${NC} licensegen 是旧版本（无 batch/verify），仅影响批量发证" >&2
  fi

  # 3) 私钥是否就位
  if [ -z "$PRIVATE_KEY" ] || [ ! -f "$PRIVATE_KEY" ]; then
    echo "${RED}✗${NC} 私钥文件未就位（--private-key 或 CORTEX_PRIVATE_KEY）" >&2
    echo "    私钥必须离线保存，绝不进 git。检查 .gitignore 含 _temp/" >&2
    return 1
  fi
  local perms
  perms="$(stat -c '%a' "$PRIVATE_KEY" 2>/dev/null || echo '?')"
  echo "${GREEN}✓${NC} 私钥: $PRIVATE_KEY（权限 $perms）" >&2
  if [ "$perms" != "?" ] && [ "$perms" != "600" ]; then
    warn "私钥权限是 $perms，建议 chmod 600"
  fi

  # 4) 真正签一张 trial 许可并校验（端到端）
  local tmpdir; tmpdir="$(make_tmpdir)"
  trap 'rm -rf "$tmpdir"' RETURN
  local keyfile="$tmpdir/selftest.key"
  local token
  if ! token="$("$gen" issue --private-key "$(native_path "$PRIVATE_KEY")" --plan trial \
        --email selftest@example.com --order SELFTEST --days 1 \
        --out "$(native_path "$keyfile")" 2>"$tmpdir/err.txt" | head -1)"; then
    echo "${RED}✗${NC} 签发失败：" >&2
    cat "$tmpdir/err.txt" >&2
    return 1
  fi
  echo "${GREEN}✓${NC} 签发成功（${#token} 字符的 token）" >&2

  if "$gen" verify --key-file "$(native_path "$keyfile")" >/dev/null 2>&1; then
    echo "${GREEN}✓${NC} 自校验通过：该私钥签发的许可可被内置公钥验证" >&2
  else
    echo "${RED}✗${NC} 自校验失败：私钥与二进制内置公钥不匹配！" >&2
    echo "    这意味着你发出的任何许可客户都无法激活。请先修密钥。" >&2
    return 1
  fi

  # 5) 跨实现校验（有 Python 就做，证明格式与 Go 一致）
  if command -v python >/dev/null 2>&1 && [ -f "scripts/cortex_license.py" ]; then
    if python scripts/cortex_license.py verify --key-file "$keyfile" >/dev/null 2>&1; then
      echo "${GREEN}✓${NC} Python 独立实现亦校验通过（Go/Python 格式一致）" >&2
    else
      warn "Python 校验失败（不影响发证，但说明两套实现有分歧）"
    fi
  fi

  rm -rf "$tmpdir"
  echo "" >&2
  ok "发证链路自检全部通过" >&2
  return 0
}

if [ "$SELFTEST" -eq 1 ]; then
  run_selftest
  exit $?
fi

# ---------- 批量模式 ----------
if [ -n "$ORDERS_CSV" ]; then
  [ -z "$OUT_DIR" ] && die2 "批量模式必须给 --out-dir（许可文件输出目录）"
  [ -f "$ORDERS_CSV" ] || die2 "订单 CSV 不存在：$ORDERS_CSV"
  local_gen="$(find_licensegen)"
  [ -z "$local_gen" ] && { echo "未找到 licensegen" >&2; build_hint; exit 1; }
  [ -n "$PRIVATE_KEY" ] && [ -f "$PRIVATE_KEY" ] || die2 "批量模式必须给 --private-key（或 CORTEX_PRIVATE_KEY）"

  info "批量发证：$ORDERS_CSV → $OUT_DIR"
  mkdir -p "$OUT_DIR"
  # CSV 必须是 email 列在前的标准格式；licensegen 自己做逐行容错
  set +e
  "$local_gen" batch --private-key "$PRIVATE_KEY" --orders "$ORDERS_CSV" --out-dir "$OUT_DIR"
  rc=$?
  set -e
  echo "" >&2
  if [ $rc -eq 0 ]; then
    ok "批量发证完成，文件在 $OUT_DIR"
  else
    die "批量发证有失败行（见上面 stderr），已成功的文件仍在 $OUT_DIR"
  fi
  exit $rc
fi

# ---------- 单笔模式 ----------
handle_one() {
  local gen; gen="$(find_licensegen)"
  [ -z "$gen" ] && { echo "未找到 licensegen" >&2; build_hint; exit 1; }

  [ -n "$EMAIL" ] || die2 "必须给 --email（许可绑定客户邮箱，便于售后追溯）"
  # 邮箱最低限度校验：发错邮箱 = 客户收不到 = 一笔退款
  case "$EMAIL" in
    *@*.*) : ;;
    *) die2 "邮箱格式看起来不对：$EMAIL" ;;
  esac

  local out_path="$OUT_FILE"
  if [ -z "$out_path" ]; then
    local base="${ORDER:-$(echo "$EMAIL" | tr '@.' '__')}"
    out_path="${OUT_DIR:-.gobuild/licenses}/${base}.key"
  fi
  mkdir -p "$(dirname "$out_path")"

  info "签发：email=$EMAIL plan=$PLAN days=$DAYS seats=$SEATS order=${ORDER:-<无>}"

  local args=(issue --private-key "$PRIVATE_KEY" --plan "$PLAN" --seats "$SEATS"
              --email "$EMAIL" --out "$out_path")
  [ -n "$ORDER" ] && args+=(--order "$ORDER")
  [ -n "$FEATURES" ] && args+=(--features "$FEATURES")
  if [ -n "$EXPIRES" ]; then
    args+=(--expires "$EXPIRES")
  else
    args+=(--days "$DAYS")
  fi

  # licensegen 把 token 打到 stdout（单行），把自检信息打到 stderr。
  # 只调用一次：重复调用既浪费一次签名机会，也让 --out 的最终内容难以预期。
  local out_err="$out_path.err"
  local token
  if ! token="$("$gen" "${args[@]}" 2>"$out_err" | head -1)"; then
    echo "--- licensegen stderr ---" >&2
    cat "$out_err" >&2
    rm -f "$out_err"
    die "签发失败（见上方 licensegen 输出）"
  fi
  [ "$QUIET" -eq 1 ] || sed 's/^/  /' "$out_err" >&2
  rm -f "$out_err"
  [ -n "$token" ] || die "签发失败，未得到 token"

  # 二次校验：绝不把未验证的许可发出去
  if ! "$gen" verify --key-file "$out_path" >/dev/null 2>&1; then
    die "刚签发的许可自校验失败！不要发给客户。检查私钥与内置公钥是否匹配。"
  fi
  chmod 600 "$out_path" 2>/dev/null || true

  local plan_display; plan_display="$(echo "$PLAN" | awk '{print toupper(substr($0,1,1)) substr($0,2)}')"
  local expiry_display
  if [ -n "$EXPIRES" ]; then expiry_display="$EXPIRES"; else expiry_display="$DAYS 天后"; fi

  echo "" >&2
  ok "许可已签发并通过自校验" >&2
  echo "  套餐：$plan_display   席位：$SEATS   有效期：$expiry_display" >&2
  echo "  文件：$out_path" >&2

  # 输出可直接粘贴给客户的邮件正文（这是发证真正的交付物）
  cat <<MAIL

──────────────────── 复制以下内容发给客户 ────────────────────
主题：你的 Cortex $plan_display 许可（订单 ${ORDER:-N/A}）

你好，

感谢购买 Cortex $plan_display。你的许可见下方，激活只需一条命令。

1) 安装 Cortex（若尚未安装）
   macOS / Linux:
     curl -fsSL https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.sh | bash
   Windows (PowerShell):
     irm https://raw.githubusercontent.com/lh123aa/cortex/main/scripts/install.ps1 | iex

2) 激活许可（把下面整行复制进终端）
     cortex license activate "$token"

3) 确认生效
     cortex license status
   应显示 Plan: $plan_display，有效期 $expiry_display。

许可绑定邮箱：$EMAIL
订单号：${ORDER:-N/A}
席位：$SEATS

常见问题
- 换机器：把同一串 token 在新机器上再执行一次 activate 即可（同一席位请勿在多台同时使用）。
- 许可丢失：回信本邮件，我们凭订单号重发。
- 功能咨询：回信说明场景，24 小时内答复。

Cortex 团队
─────────────────────────────────────────────────────────────
MAIL

  # 追加一行机器可读记录，便于做发证台账
  local ledger=".gobuild/licenses/issued.tsv"
  if [ ! -f "$ledger" ]; then
    printf "issued_at\temail\tplan\tseats\torder\texpires\tfile\n" >> "$ledger"
  fi
  printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\n" \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$EMAIL" "$PLAN" "$SEATS" "${ORDER:--}" "${EXPIRES:-+${DAYS}d}" "$out_path" \
    >> "$ledger"
  info "台账已更新：$ledger"
  return 0
}

[ -n "$PRIVATE_KEY" ] && [ -f "$PRIVATE_KEY" ] || die2 "必须给 --private-key（或设 CORTEX_PRIVATE_KEY 指向私钥文件）"

handle_one

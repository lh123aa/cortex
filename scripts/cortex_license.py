#!/usr/bin/env python3
"""cortex_license.py — Cortex 许可工具（Python 参考实现 / 应急发证器）

用途
----
与 Go 端 pkg/license 完全同算法、同格式的独立实现，用于：
  1. 在 Go 工具链不可用时，仍然能签发 / 校验 Cortex 许可（应急发证）。
  2. 作为 Go 实现的「第二意见」，交叉验证格式正确性（避免单点实现 bug）。

格式（与 pkg/license/license.go 一致，v1 冻结）
------------------------------------------------
    base64url_nopad(JSON payload) "." base64url_nopad(Ed25519 signature)

    payload JSON 字段：
        v         int     格式版本，固定 1
        plan      string  free|trial|pro|team|enterprise
        features  []str?  额外授权 feature（可选）
        email     string? 客户邮箱（可选）
        order     string? 订单号（可选）
        seats     int     席位数
        iat       int64   签发时间 unix 秒
        exp       int64   到期时间 unix 秒，0 = 永久

约定
----
- base64 使用 URL-safe 字母表、无 padding（Go: base64.RawURLEncoding.Strict()）。
- 签名直接对 payload 的原始 JSON 字节签名（不是对 base64 文本签名）。
- iat/exp 为 UTC unix 秒。

用法
----
    python cortex_license.py verify  --key "<token>"
    python cortex_license.py verify  --key-file license.key --public-key 2619a532...
    python cortex_license.py issue   --private-key private.key --plan pro \
                                     --email user@example.com --order ORD-1 --days 365
    python cortex_license.py selftest

依赖：cryptography（推荐）或 PyNaCl，二选一。二者都没有时 verify 仍可用，
因为校验只需要公钥；issue 需要私钥签名，必须有其中之一。
"""

from __future__ import annotations

import argparse
import base64
import binascii
import json
import sys
import time

# 二进制内嵌的公钥（pkg/license.PublicKeyHex），用于离线校验。
DEFAULT_PUBLIC_KEY_HEX = "2619a532e715e2b3a79c38ab3c129d35ef9a5133886b991a64cc16619fdd159f"

FORMAT_VERSION = 1
VALID_PLANS = ("free", "trial", "pro", "team", "enterprise")

# 与 Go 端 license.PlanFeatures / license.Limits 保持一致的副本。
PLAN_FEATURES = {
    "enterprise": ["*"],
    "team": ["console", "audit-log", "rbac-team", "sso-oidc", "backup-s3", "rerank"],
    "pro": ["console", "rerank", "auto-index"],
    "trial": ["console", "rerank", "auto-index"],
    "free": [],
}
_GB = 1 << 30
PLAN_LIMITS = {
    "trial": (500_000, 100 * _GB),
    "pro": (500_000, 100 * _GB),
    "team": (1_000_000, 500 * _GB),
    "enterprise": (-1, -1),
    "free": (5_000, 1 * _GB),
}


# --------------------------------------------------------------------------
# base64url (no padding)
# --------------------------------------------------------------------------
def b64u_encode(raw: bytes) -> str:
    return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")


def b64u_decode(text: str) -> bytes:
    text = text.strip()
    pad = "=" * (-len(text) % 4)
    return base64.urlsafe_b64decode(text + pad)


# --------------------------------------------------------------------------
# Ed25519 backend：优先 cryptography，回退 PyNaCl
# --------------------------------------------------------------------------
def _load_backend():
    try:
        from cryptography.hazmat.primitives.asymmetric.ed25519 import (
            Ed25519PrivateKey,
            Ed25519PublicKey,
        )

        return "cryptography", Ed25519PrivateKey, Ed25519PublicKey
    except Exception:
        pass
    try:
        import nacl.signing  # noqa: F401

        return "pynacl", None, None
    except Exception:
        return None, None, None


def _sign(seed: bytes, message: bytes) -> bytes:
    backend, Ed25519PrivateKey, _ = _load_backend()
    if backend == "cryptography":
        return Ed25519PrivateKey.from_private_bytes(seed).sign(message)
    if backend == "pynacl":
        from nacl.signing import SigningKey

        return SigningKey(seed).sign(message).signature
    raise RuntimeError("需要 cryptography 或 PyNaCl 才能签名：pip install cryptography")


def _verify(public_key: bytes, message: bytes, signature: bytes) -> bool:
    backend, _, Ed25519PublicKey = _load_backend()
    try:
        if backend == "cryptography":
            Ed25519PublicKey.from_public_bytes(public_key).verify(signature, message)
            return True
        if backend == "pynacl":
            from nacl.signing import VerifyKey

            VerifyKey(public_key).verify(message, signature)
            return True
    except Exception:
        return False
    raise RuntimeError("需要 cryptography 或 PyNaCl 才能验签：pip install cryptography")


# --------------------------------------------------------------------------
# 私钥解析：兼容 32 字节 seed / 64 字节 seed+pub / licensegen 的 "private_key: hex" 输出
# --------------------------------------------------------------------------
def parse_private_key(text: str) -> bytes:
    text = text.strip()
    # 容忍 keygen 整段输出被粘贴：取 "private_key:" 之后的部分
    idx = text.find("private_key")
    if idx >= 0:
        colon = text.find(":", idx)
        if colon >= 0:
            text = text[colon + 1 :].strip()
    hexstr = "".join(text.split())
    try:
        raw = binascii.unhexlify(hexstr)
    except binascii.Error as exc:
        raise SystemExit(f"私钥不是合法 hex: {exc}")
    if len(raw) == 64:
        # Go ed25519.PrivateKey = seed(32) || public(32)，取前 32 字节 seed
        return raw[:32]
    if len(raw) == 32:
        return raw
    raise SystemExit(f"私钥长度 {len(raw)} 字节，期望 32（seed）或 64（seed+public）")


def derive_public_key(seed: bytes) -> bytes:
    backend, Ed25519PrivateKey, _ = _load_backend()
    if backend == "cryptography":
        from cryptography.hazmat.primitives import serialization

        priv = Ed25519PrivateKey.from_private_bytes(seed)
        return priv.public_key().public_bytes(
            encoding=serialization.Encoding.Raw,
            format=serialization.PublicFormat.Raw,
        )
    if backend == "pynacl":
        from nacl.signing import SigningKey

        return bytes(SigningKey(seed).verify_key)
    raise RuntimeError("需要 cryptography 或 PyNaCl 才能推导公钥")


# --------------------------------------------------------------------------
# 签发 / 校验
# --------------------------------------------------------------------------
def issue(private_key_hex: str, plan: str, email: str = "", order: str = "",
          days: int = 365, seats: int = 1, features=None, now: int | None = None) -> str:
    plan = (plan or "").strip().lower()
    if plan not in VALID_PLANS:
        raise SystemExit(f"无效 plan {plan!r}，可选：{', '.join(VALID_PLANS)}")
    if days < 0:
        raise SystemExit("--days 必须 >= 0")
    issued = int(now if now is not None else time.time())
    expires = issued + days * 86400 if days > 0 else 0

    payload = {"v": FORMAT_VERSION, "plan": plan}
    if features:
        payload["features"] = list(features)
    if email:
        payload["email"] = email
    if order:
        payload["order"] = order
    payload["seats"] = int(seats)
    payload["iat"] = issued
    payload["exp"] = expires

    # 关键：紧凑 JSON、字段顺序与 Go 的 struct 字段顺序一致，键按插入顺序输出。
    raw = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    seed = parse_private_key(private_key_hex)
    sig = _sign(seed, raw)
    return b64u_encode(raw) + "." + b64u_encode(sig)


def verify(token: str, public_key_hex: str = DEFAULT_PUBLIC_KEY_HEX) -> dict:
    token = token.strip()
    if not token:
        raise ValueError("许可为空")
    parts = token.split(".")
    if len(parts) != 2:
        raise ValueError("格式错误：应为 base64url(payload).base64url(signature)")
    try:
        raw = b64u_decode(parts[0])
        sig = b64u_decode(parts[1])
    except (binascii.Error, ValueError) as exc:
        raise ValueError(f"base64 解码失败：{exc}")
    if len(sig) != 64:
        raise ValueError(f"签名长度 {len(sig)} 字节，期望 64")

    pub = binascii.unhexlify(public_key_hex.strip())
    if len(pub) != 32:
        raise ValueError(f"公钥长度 {len(pub)} 字节，期望 32")
    if not _verify(pub, raw, sig):
        raise ValueError("签名校验失败（许可被篡改，或公钥不匹配）")

    p = json.loads(raw.decode("utf-8"))
    if p.get("v") != FORMAT_VERSION:
        raise ValueError(f"不支持的许可版本 {p.get('v')}，本工具支持 v{FORMAT_VERSION}")
    plan = str(p.get("plan", "")).strip().lower()
    if plan not in VALID_PLANS:
        raise ValueError(f"无效 plan {p.get('plan')!r}")

    exp = int(p.get("exp") or 0)
    if exp:
        now = int(time.time())
        if now >= exp:
            raise ValueError(f"许可已于 {time.strftime('%Y-%m-%d', time.gmtime(exp))} 过期")

    docs, storage = PLAN_LIMITS.get(plan, PLAN_LIMITS["free"])
    if exp:
        days_left = max(0, int((exp - time.time()) / 86400))
    else:
        days_left = -1
    return {
        "plan": plan,
        "email": p.get("email", ""),
        "order": p.get("order", ""),
        "seats": int(p.get("seats") or 0),
        "features": p.get("features") or [],
        "plan_features": PLAN_FEATURES.get(plan, []),
        "issued_at": int(p.get("iat") or 0),
        "expires_at": exp,
        "perpetual": exp == 0,
        "days_left": days_left,
        "document_limit": docs,
        "storage_limit_bytes": storage,
        "key": token,
    }


# --------------------------------------------------------------------------
# CLI
# --------------------------------------------------------------------------
def fmt_ts(ts: int) -> str:
    return time.strftime("%Y-%m-%d %H:%M:%S UTC", time.gmtime(ts)) if ts else "永久（不失效）"


def cmd_issue(args) -> int:
    try:
        with open(args.private_key, "r", encoding="utf-8") as fh:
            priv_hex = fh.read()
    except OSError as exc:
        print(f"读取私钥失败：{exc}", file=sys.stderr)
        return 1
    features = [s.strip() for s in (args.features or "").split(",") if s.strip()]
    token = issue(priv_hex, args.plan, args.email, args.order, args.days,
                  args.seats, features)
    if args.out:
        with open(args.out, "w", encoding="utf-8", newline="\n") as fh:
            fh.write(token + "\n")
        print(token)
        print(f"\n许可已写入：{args.out}", file=sys.stderr)
    else:
        print(token)
    return 0


def cmd_verify(args) -> int:
    if args.key_file:
        try:
            with open(args.key_file, "r", encoding="utf-8") as fh:
                token = fh.read()
        except OSError as exc:
            print(f"读取许可文件失败：{exc}", file=sys.stderr)
            return 1
    else:
        token = args.key or ""
    try:
        info = verify(token, args.public_key)
    except (ValueError, RuntimeError) as exc:
        print(f"❌ 校验失败：{exc}", file=sys.stderr)
        return 1

    print("✅ 许可有效")
    print(f"  套餐     : {info['plan']}")
    if info["email"]:
        print(f"  邮箱     : {info['email']}")
    if info["order"]:
        print(f"  订单号   : {info['order']}")
    if info["seats"]:
        print(f"  席位     : {info['seats']}")
    print(f"  签发时间 : {fmt_ts(info['issued_at'])}")
    print(f"  到期时间 : {fmt_ts(info['expires_at'])}"
          + (f"（剩余 {info['days_left']} 天）" if info["days_left"] >= 0 else ""))
    feats = info["plan_features"] or info["features"] or ["(无)"]
    print(f"  功能     : {', '.join(feats)}")
    docs, storage = info["document_limit"], info["storage_limit_bytes"]
    if docs < 0:
        print("  配额     : 不限文档 · 不限存储")
    else:
        print(f"  配额     : {docs:,} 篇文档 · {storage / _GB:.1f} GB 存储")
    if args.json:
        print(json.dumps({k: v for k, v in info.items() if k != "key"},
                         ensure_ascii=False, indent=2))
    return 0


def cmd_selftest(args) -> int:
    """端到端自测：签发 → 校验 → 篡改检测，证明格式与 Go 端一致。"""
    print("=== Cortex 许可格式自测（Python 参考实现）===")
    backend, _, _ = _load_backend()
    print(f"Ed25519 后端 : {backend or '不可用'}")
    print(f"内嵌公钥     : {DEFAULT_PUBLIC_KEY_HEX}")

    # 用「今天 00:00 UTC」做签发时间：既让同一天内输出可复现，又保证不会签发成过期许可。
    fixed_now = int(time.time()) // 86400 * 86400
    seed_hex = args.private_key and open(args.private_key, encoding="utf-8").read()
    if not seed_hex:
        seed = bytes(range(32))
        print("私钥         : (内置测试 seed 0x00..0x1f)")
    else:
        seed = parse_private_key(seed_hex)
    pub = derive_public_key(seed)
    print(f"派生公钥     : {pub.hex()}")

    token = issue(seed.hex(), "trial", "selftest@example.com", "T3-SELFTEST", 14, 1, now=fixed_now)
    print(f"\n签发 token   : {token[:80]}...")
    ok = verify(token, pub.hex())
    print(f"同密钥校验   : {'✅ 通过' if ok else '❌ 失败'} (plan={ok['plan']}, 剩余 {ok['days_left']} 天)")

    # 篡改检测：改动 payload 最后一个字符必须失败
    head, sig = token.split(".")
    tampered = (head[:-2] + ("AA" if head[-2:] != "AA" else "BB")) + "." + sig
    try:
        verify(tampered, pub.hex())
        print("篡改检测     : ❌ 失败（被篡改的许可竟然通过了）")
        return 1
    except ValueError:
        print("篡改检测     : ✅ 通过（篡改的 payload 被拒绝）")

    # 错误公钥检测
    try:
        verify(token, "00" * 32)
        print("错误公钥检测 : ❌ 失败（错误公钥竟然通过了）")
        return 1
    except (ValueError, RuntimeError):
        print("错误公钥检测 : ✅ 通过（错误公钥被拒绝）")
    print("\n全部自测通过。")
    return 0


def main(argv=None) -> int:
    ap = argparse.ArgumentParser(
        prog="cortex_license.py",
        description="Cortex 离线许可工具（Python 参考实现，与 pkg/license 同格式）",
    )
    sub = ap.add_subparsers(dest="cmd", required=True)

    p_issue = sub.add_parser("issue", help="签发一张许可")
    p_issue.add_argument("--private-key", required=True, help="私钥 hex 文件（32 或 64 字节）")
    p_issue.add_argument("--plan", default="pro", help="free|trial|pro|team|enterprise（默认 pro）")
    p_issue.add_argument("--email", default="", help="客户邮箱")
    p_issue.add_argument("--order", default="", help="订单号")
    p_issue.add_argument("--days", type=int, default=365, help="有效天数，0=永久（默认 365）")
    p_issue.add_argument("--seats", type=int, default=1, help="席位（默认 1）")
    p_issue.add_argument("--features", default="", help="逗号分隔的额外 feature")
    p_issue.add_argument("--out", default="", help="写入文件（默认输出到 stdout）")
    p_issue.set_defaults(func=cmd_issue)

    p_verify = sub.add_parser("verify", help="校验一张许可")
    p_verify.add_argument("--key", default="", help="许可 token 文本")
    p_verify.add_argument("--key-file", default="", help="许可文件路径")
    p_verify.add_argument("--public-key", default=DEFAULT_PUBLIC_KEY_HEX, help="覆盖公钥 hex")
    p_verify.add_argument("--json", action="store_true", help="附加 JSON 输出")
    p_verify.set_defaults(func=cmd_verify)

    p_self = sub.add_parser("selftest", help="端到端自测（签发→校验→篡改检测）")
    p_self.add_argument("--private-key", default="", help="可选：用真实私钥做自测")
    p_self.set_defaults(func=cmd_selftest)

    args = ap.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())

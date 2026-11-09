#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""GitHub 指标快照 —— 把「发布到底有没有效果」变成可测量的时间序列。

为什么需要它：Cortex 当前 0 曝光、0 咨询、0 付费，唯一的判断依据是
「发出去之后 star / 下载 / issue 有没有动」。没有时间序列，就只能靠感觉。

数据源（按可用性降级，永不失败）：
  1) shields.io 徽章 JSON  —— 无速率限制，稳定，覆盖 stars/forks/issues/downloads
  2) GitHub REST API       —— 补充 latest_tag / pushed_at（未认证 60 次/小时）
  3) git ls-remote         —— 判断本地提交有没有推上去（不耗 API 配额）

用法：
  python scripts/track_metrics.py            # 采集一次，追加到 project-journal/metrics/github.csv
  python scripts/track_metrics.py --show     # 只看历史趋势
  python scripts/track_metrics.py --json     # 机器可读（给 journal.py metric 喂数）
  python scripts/track_metrics.py --no-net   # 只读本地 CSV

退出码：0 成功；2 采集失败（网络全断）。
"""
import argparse
import csv
import json
import os
import re
import subprocess
import sys
import urllib.request
import urllib.error
from datetime import datetime, timezone

REPO = os.environ.get("CORTEX_REPO", "lh123aa/cortex")
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CSV_PATH = os.path.join(ROOT, "project-journal", "metrics", "github.csv")
FIELDS = ["date", "stars", "forks", "open_issues", "downloads_total",
          "downloads_v401", "latest_tag", "head_pushed", "source"]


def http_json(url, timeout=20):
    req = urllib.request.Request(url, headers={"User-Agent": "cortex-metrics/1.1",
                                               "Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode("utf-8"))


def shield(path):
    """shields.io 徽章 JSON -> 数字；失败返回 None。path 形如 stars、downloads/{owner}/{repo}/total。"""
    try:
        j = http_json("https://img.shields.io/github/%s.json" % path)
        v = j.get("value") or j.get("message") or ""
        m = re.search(r"(\d[\d,]*)", str(v))
        return int(m.group(1).replace(",", "")) if m else None
    except Exception:
        return None


def git_remote_head():
    try:
        out = subprocess.run(["git", "ls-remote", "origin", "HEAD"],
                             cwd=ROOT, capture_output=True, text=True, timeout=25)
        parts = out.stdout.split()
        return parts[0][:7] if parts else ""
    except Exception:
        return ""


def local_head():
    try:
        out = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT,
                             capture_output=True, text=True, timeout=15)
        return out.stdout.strip()[:7]
    except Exception:
        return ""


def collect():
    src = []
    stars = shield("stars/" + REPO)
    forks = shield("forks/" + REPO)
    issues = shield("issues/" + REPO)
    dl_total = shield("downloads/%s/total" % REPO)
    if stars is not None:
        src.append("shields")

    tag, pushed_at = "", ""
    try:
        repo = http_json("https://api.github.com/repos/" + REPO)
        stars = repo.get("stargazers_count", stars)
        forks = repo.get("forks_count", forks)
        issues = repo.get("open_issues_count", issues)
        pushed_at = repo.get("pushed_at", "")
        if "api" not in src:
            src.append("api")
    except Exception:
        pass

    dl401 = None
    try:
        j = http_json("https://img.shields.io/github/downloads/%s/v4.0.1/total.json" % REPO)
        m = re.search(r"(\d[\d,]*)", str(j.get("value") or j.get("message") or ""))
        if m:
            dl401 = int(m.group(1).replace(",", ""))
    except Exception:
        pass

    lh, rh = local_head(), git_remote_head()
    head_pushed = "unknown" if not rh else ("yes" if lh == rh else "no")

    if stars is None and dl_total is None:
        raise RuntimeError("所有数据源都不可用（网络或代理问题）")

    return {
        "date": datetime.now(timezone.utc).astimezone().strftime("%Y-%m-%d %H:%M"),
        "stars": stars if stars is not None else "",
        "forks": forks if forks is not None else "",
        "open_issues": issues if issues is not None else "",
        "downloads_total": dl_total if dl_total is not None else "",
        "downloads_v401": dl401 if dl401 is not None else "",
        "latest_tag": tag,
        "head_pushed": head_pushed,
        "source": "+".join(src) or "partial",
    }


def read_rows():
    if not os.path.exists(CSV_PATH):
        return []
    with open(CSV_PATH, newline="", encoding="utf-8") as f:
        return [r for r in csv.DictReader(f) if r.get("date")]


def append(row):
    os.makedirs(os.path.dirname(CSV_PATH), exist_ok=True)
    new = not os.path.exists(CSV_PATH)
    with open(CSV_PATH, "a", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=FIELDS)
        if new:
            w.writeheader()
        w.writerow(row)


def num(x):
    try:
        return int(x)
    except (TypeError, ValueError):
        return None


def show():
    rows = read_rows()
    if not rows:
        print("还没有快照。先跑一次：python scripts/track_metrics.py")
        return
    print("%-17s %5s %5s %5s %9s %9s %9s  %s" %
          ("时间", "star", "fork", "iss", "下载累计", "starΔ", "下载Δ", "已推送"))
    prev = None
    for r in rows:
        ds = dd = ""
        if prev:
            a, b = num(r["stars"]), num(prev["stars"])
            if a is not None and b is not None:
                ds = "%+d" % (a - b)
            a, b = num(r["downloads_total"]), num(prev["downloads_total"])
            if a is not None and b is not None:
                dd = "%+d" % (a - b)
        print("%-17s %5s %5s %5s %9s %9s %9s  %s" %
              (r["date"], r["stars"], r["forks"], r["open_issues"],
               r["downloads_total"], ds, dd, r["head_pushed"]))
        prev = r
    if len(rows) >= 2:
        f, l = rows[0], rows[-1]
        a, b = num(l["stars"]), num(f["stars"])
        c, d = num(l["downloads_total"]), num(f["downloads_total"])
        print("\n自 %s 起：star %s，下载 %s" % (
            f["date"],
            "%+d" % (a - b) if None not in (a, b) else "n/a",
            "%+d" % (c - d) if None not in (c, d) else "n/a"))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--show", action="store_true", help="只显示历史，不联网")
    ap.add_argument("--json", action="store_true", help="输出 JSON 不写盘")
    ap.add_argument("--no-net", action="store_true", help="等价于 --show")
    a = ap.parse_args()
    if a.show or a.no_net:
        show()
        return 0
    try:
        row = collect()
    except Exception as e:
        print("采集失败：%s" % e, file=sys.stderr)
        return 2
    if a.json:
        print(json.dumps(row, ensure_ascii=False))
        return 0
    append(row)
    print("已记录  star=%s 下载=%s (v4.0.1=%s) 已推送=%s 源=%s" %
          (row["stars"], row["downloads_total"], row["downloads_v401"],
           row["head_pushed"], row["source"]))
    show()
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
检查「合并库」是否存在且包含 messages 表；若缺失或比源库旧则调用 dingwave -merged-out -export-only 再生成。

使用前请配置环境变量（或由命令行传入等价参数）：
  DINGWAVE_SOURCE_DB   — 与 dingwave -d 相同（加密库路径或已解密路径）
  DINGWAVE_EXTRA_FLAGS — 可选，须拆成多参数时本脚本用 shlex 解析，例如：-k 你的uid -salt 你的salt
  DINGWAVE_MERGED_DB   — 合并库输出路径；默认为本技能目录下 cache/merged.db
  DINGWAVE_BIN         — dingwave 可执行文件；默认在仓库根查找名为 dingwave 的文件，否则用 PATH 里的 dingwave
"""

from __future__ import annotations

import argparse
import os
import shlex
import sqlite3
import subprocess
import sys
from pathlib import Path


def skill_root() -> Path:
    # scripts/ensure_merged.py -> 技能根目录 dingwave-messages/
    return Path(__file__).resolve().parents[1]


def default_merged_path() -> Path:
    d = skill_root() / "cache"
    d.mkdir(parents=True, exist_ok=True)
    return d / "merged.db"


def find_repo_root(start: Path) -> Path | None:
    """向上找到含有 go.mod 的目录，作为 dingwave 仓库根。"""
    p = start.resolve()
    for _ in range(12):
        if (p / "go.mod").is_file():
            return p
        if p == p.parent:
            break
        p = p.parent
    return None


def resolve_dingwave_bin(explicit: str | None) -> str:
    if explicit:
        return explicit
    env = os.environ.get("DINGWAVE_BIN")
    if env:
        return env
    root = find_repo_root(skill_root())
    if root:
        cand = root / "dingwave"
        if cand.is_file() and os.access(cand, os.X_OK):
            return str(cand)
    return "dingwave"


def merged_is_valid(merged: Path) -> bool:
    if not merged.is_file():
        return False
    try:
        conn = sqlite3.connect(f"file:{merged.resolve()}?mode=ro", uri=True)
        try:
            row = conn.execute(
                "SELECT 1 FROM sqlite_master WHERE type='table' AND name='messages' LIMIT 1"
            ).fetchone()
            return row is not None
        finally:
            conn.close()
    except sqlite3.Error:
        return False


def needs_rebuild(merged: Path, source: Path | None) -> bool:
    """缺少有效合并库，或源库更新比合并库新时返回 True。"""
    if not merged_is_valid(merged):
        return True
    if source is None or not source.is_file():
        return False
    try:
        return source.stat().st_mtime > merged.stat().st_mtime
    except OSError:
        return True


def extra_argv_from_env() -> list[str]:
    raw = os.environ.get("DINGWAVE_EXTRA_FLAGS", "")
    if not raw.strip():
        return []
    return shlex.split(raw)


def run_export(bin_path: str, source_db: str, merged_db: str, extra: list[str]) -> None:
    """调用 dingwave：解密/迁移逻辑与起服务相同，仅写入合并库后退出。"""
    cmd = [bin_path, "-d", source_db, *extra, "-merged-out", merged_db, "-export-only"]
    print("执行:", " ".join(shlex.quote(c) for c in cmd), file=sys.stderr)
    subprocess.run(cmd, check=True)


def main() -> int:
    parser = argparse.ArgumentParser(description="确保 Dingwave 合并库存在且较新")
    parser.add_argument(
        "--merged",
        type=str,
        default=None,
        help="合并库路径（默认同 DINGWAVE_MERGED_DB 或 cache/merged.db）",
    )
    parser.add_argument(
        "--source",
        type=str,
        default=None,
        help="源库 -d（默认同 DINGWAVE_SOURCE_DB）",
    )
    parser.add_argument(
        "--bin",
        type=str,
        default=None,
        help="dingwave 可执行文件路径",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="忽略新鲜度，强制重新导出",
    )
    args = parser.parse_args()

    merged = Path(args.merged or os.environ.get("DINGWAVE_MERGED_DB") or default_merged_path())
    merged.parent.mkdir(parents=True, exist_ok=True)

    src_raw = args.source or os.environ.get("DINGWAVE_SOURCE_DB")
    source = Path(src_raw) if src_raw else None

    if args.force:
        rebuild = True
    else:
        rebuild = needs_rebuild(merged, source)

    if not rebuild:
        print(f"合并库已就绪: {merged.resolve()}", file=sys.stderr)
        print(str(merged.resolve()))
        return 0

    if not src_raw:
        print(
            "需要重新生成合并库，但未设置源库路径。请设置环境变量 DINGWAVE_SOURCE_DB\n"
            "或传入 --source，并配置解密相关参数 DINGWAVE_EXTRA_FLAGS（与命令行 -k/-salt/-userconfig 一致）。",
            file=sys.stderr,
        )
        return 2

    bin_path = resolve_dingwave_bin(args.bin)
    extra = extra_argv_from_env()
    try:
        run_export(bin_path, src_raw, str(merged.resolve()), extra)
    except FileNotFoundError:
        print(f"找不到可执行文件: {bin_path}", file=sys.stderr)
        return 127
    except subprocess.CalledProcessError as e:
        print(f"dingwave 退出码 {e.returncode}", file=sys.stderr)
        return e.returncode or 1

    if not merged_is_valid(merged):
        print("导出后仍无法打开有效的 messages 表", file=sys.stderr)
        return 1

    print(f"合并库已更新: {merged.resolve()}", file=sys.stderr)
    print(str(merged.resolve()))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

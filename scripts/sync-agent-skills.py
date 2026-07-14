#!/usr/bin/env python3
"""Synchronize canonical Agent Skills into Claude Code's project skill directory."""

from __future__ import annotations

import argparse
import shutil
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / ".agents" / "skills"
TARGET = ROOT / ".claude" / "skills"
NOTICE = "<!-- GENERATED MIRROR. Edit .agents/skills and run scripts/sync-agent-skills.py. -->\n"


def expected() -> dict[Path, str]:
    files: dict[Path, str] = {}
    for source in sorted(SOURCE.rglob("*")):
        if not source.is_file():
            continue
        relative = source.relative_to(SOURCE)
        content = source.read_text(encoding="utf-8")
        if source.name == "SKILL.md":
            content = NOTICE + content
        files[relative] = content
    return files


def check() -> int:
    wanted = expected()
    actual = (
        {path.relative_to(TARGET) for path in TARGET.rglob("*") if path.is_file()}
        if TARGET.exists()
        else set()
    )

    failures: list[str] = []
    for path in sorted(set(wanted) - actual):
        failures.append(f"missing: {path}")
    for path in sorted(actual - set(wanted)):
        failures.append(f"extra: {path}")
    for relative, content in wanted.items():
        target = TARGET / relative
        if not target.exists() or target.read_text(encoding="utf-8") != content:
            failures.append(f"out of sync: {relative}")

    if failures:
        print("Claude skill mirrors are out of sync:")
        for failure in failures:
            print(f"  - {failure}")
        return 1

    print("Claude skill mirrors are synchronized.")
    return 0


def sync() -> int:
    if TARGET.exists():
        shutil.rmtree(TARGET)
    TARGET.mkdir(parents=True, exist_ok=True)
    for relative, content in expected().items():
        target = TARGET / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
    print(f"Synchronized {SOURCE} -> {TARGET}")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    return check() if args.check else sync()


if __name__ == "__main__":
    sys.exit(main())

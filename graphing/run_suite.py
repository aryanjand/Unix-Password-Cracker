#!/usr/bin/env python3
"""Run the current controller/worker binaries and write structured JSON runs.

The Go controller writes millisecond metrics via -metrics-json. This script
only adds the suite labels (password, worker count, algorithm) and waits for
each process group to finish.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import threading
import time
from pathlib import Path


HERE = Path(__file__).resolve().parent
REPO = HERE.parent
MODULE = REPO / "distributed-multi-workers"
SHADOW_DIR = MODULE / "testdata" / "shadow"
BIN_DIR = HERE / ".bin"
RUNS_DIR = HERE / "output" / "runs"

PASSWORDS = (
    ("Ace", "shadow_ACE_bcrypt"),
    ("Bad", "shadow_BAD_bcrypt"),
    ("Cab", "shadow_CAB_bcrypt"),
    ("Dad", "shadow_DAD_bcrypt"),
    ("Ear", "shadow_EAR_bcrypt"),
)
WORKER_COUNTS = (1, 2, 3, 5)
USERNAME = "aryan"
THREADS = 4
HASH_ALGORITHM = "bcrypt"
DEFAULT_TIMEOUT_S = 180


def find_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def build_binaries() -> tuple[Path, Path]:
    BIN_DIR.mkdir(parents=True, exist_ok=True)
    controller = BIN_DIR / "controller"
    worker = BIN_DIR / "worker"
    for name, out in (("controller", controller), ("worker", worker)):
        subprocess.run(
            ["go", "build", "-o", str(out), f"./cmd/{name}"],
            cwd=MODULE,
            check=True,
        )
    return controller, worker


def pump_output(proc: subprocess.Popen, lines: list[str], ready: threading.Event) -> None:
    assert proc.stdout is not None
    for line in proc.stdout:
        sys.stdout.write(line)
        sys.stdout.flush()
        lines.append(line)
        if "listening for workers" in line:
            ready.set()
    ready.set()


def terminate(proc: subprocess.Popen | None) -> None:
    if proc is None or proc.poll() is not None:
        return
    proc.terminate()
    try:
        proc.wait(timeout=3)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait(timeout=3)


def run_one(
    controller: Path,
    worker: Path,
    label: str,
    shadow: Path,
    workers: int,
    timeout_s: int,
) -> dict:
    RUNS_DIR.mkdir(parents=True, exist_ok=True)
    slug = f"{label.lower()}_w{workers}"
    raw_path = RUNS_DIR / f"{slug}.controller.json"
    out_path = RUNS_DIR / f"{slug}.json"
    db_path = RUNS_DIR / f"{slug}.db"
    log_path = RUNS_DIR / f"{slug}.log"

    port = find_free_port()
    ctrl_cmd = [
        str(controller),
        "-p",
        str(port),
        "-f",
        str(shadow),
        "-u",
        USERNAME,
        "-b",
        "1",
        "-c",
        "1000",
        "-k",
        "100",
        "-reset",
        "-d",
        str(db_path),
        "-metrics-json",
        str(raw_path),
    ]
    worker_cmd = [
        str(worker),
        "-c",
        "127.0.0.1",
        "-p",
        str(port),
        "-t",
        str(THREADS),
    ]

    print(f"\n=== {label} × {workers} worker(s) on :{port} ===", flush=True)

    lines: list[str] = []
    ready = threading.Event()
    ctrl = subprocess.Popen(
        ctrl_cmd,
        cwd=MODULE,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        bufsize=1,
    )
    pump = threading.Thread(target=pump_output, args=(ctrl, lines, ready), daemon=True)
    pump.start()

    worker_procs: list[subprocess.Popen] = []
    try:
        if not ready.wait(timeout=20):
            raise RuntimeError("controller did not start listening")

        for _ in range(workers):
            worker_procs.append(
                subprocess.Popen(
                    worker_cmd,
                    cwd=MODULE,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                )
            )

        try:
            rc = ctrl.wait(timeout=timeout_s)
        except subprocess.TimeoutExpired as exc:
            raise RuntimeError(f"timed out after {timeout_s}s") from exc
        if rc != 0:
            raise RuntimeError(f"controller exited {rc}")
        if not raw_path.is_file():
            raise RuntimeError(f"controller did not write {raw_path}")
    except Exception:
        terminate(ctrl)
        raise
    finally:
        for proc in worker_procs:
            terminate(proc)
        pump.join(timeout=2)
        log_path.write_text("".join(lines), encoding="utf-8")

    metrics = json.loads(raw_path.read_text(encoding="utf-8"))
    record = {
        "password_label": label,
        "hash_algorithm": HASH_ALGORITHM,
        "number_of_workers": workers,
        "threads": THREADS,
        "shadow_file": str(shadow.relative_to(REPO)),
        **metrics,
    }
    out_path.write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {out_path}", flush=True)
    return record


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the graphing benchmark suite")
    parser.add_argument(
        "--passwords",
        nargs="+",
        metavar="NAME",
        help="subset of Ace Bad Cab Dad Ear (default: all)",
    )
    parser.add_argument(
        "--workers",
        nargs="+",
        type=int,
        metavar="N",
        help="worker counts (default: 1 2 3 5)",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=int(os.environ.get("GRAPH_RUN_TIMEOUT", DEFAULT_TIMEOUT_S)),
        help="per-run timeout in seconds",
    )
    return parser.parse_args()


def main() -> None:
    if shutil.which("go") is None:
        raise SystemExit("go toolchain not found on PATH")

    args = parse_args()
    wanted = {name.lower() for name in args.passwords} if args.passwords else None
    worker_counts = tuple(args.workers) if args.workers else WORKER_COUNTS

    suite = [
        (label, shadow_name)
        for label, shadow_name in PASSWORDS
        if wanted is None or label.lower() in wanted
    ]
    if not suite:
        raise SystemExit("no matching passwords")

    if RUNS_DIR.exists() and wanted is None and args.workers is None:
        for path in RUNS_DIR.iterdir():
            if path.is_file():
                path.unlink()

    controller, worker = build_binaries()
    records = []
    for label, shadow_name in suite:
        shadow = SHADOW_DIR / shadow_name
        if not shadow.is_file():
            raise SystemExit(f"missing fixture {shadow}")
        for count in worker_counts:
            records.append(
                run_one(controller, worker, label, shadow, count, args.timeout)
            )

    print(f"\nCompleted {len(records)} run(s) in {RUNS_DIR}", flush=True)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Generate recruiter-facing charts and CSVs from pasted controller metric logs.

Hash algorithm: prefer a `# algo=<name>` header in each data file, then an
algorithm token in the filename (bcrypt, sha256, sha512, md5, yescrypt).
If neither is present, default to bcrypt — the algorithm used for the
assignment measurement runs in graphing/data/.
"""

from __future__ import annotations

import json
import re
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import pandas as pd


HERE = Path(__file__).resolve().parent
REPO = HERE.parent
README_PATH = REPO / "README.md"
DATA_DIR = HERE / "data"
OUT_DIR = HERE / "output"
RUNS_DIR = OUT_DIR / "runs"

JSON_SUMMARY_FIELDS = (
    "controller_side_parsing_time_ms",
    "job_dispatch_registration_overhead_ms",
    "work_assignment_overhead_total_ms",
    "work_assignment_overhead_per_unit_ns",
    "worker_cracking_time_ms",
    "result_return_latency_ms",
    "checkpoint_overhead_ms",
    "checkpoint_impact_pct",
    "total_end_to_end_runtime_ms",
    "controller_overhead_ms",
    "networking_overhead_ms",
    "combined_overhead_ms",
    "checkpoint_observation_count",
    "checkpoint_avg_ms",
    "checkpoint_min_ms",
    "checkpoint_max_ms",
)

# Default when no `# algo=` header and no algorithm token in the filename.
DEFAULT_ALGO = "bcrypt"

PASSWORD_ORDER = ["Ace", "Bad", "Cab", "Dad", "Ear"]
PASSWORD_COLORS = {
    "Ace": "#2563eb",
    "Bad": "#dc2626",
    "Cab": "#059669",
    "Dad": "#d97706",
    "Ear": "#7c3aed",
}
OVERHEAD_COLORS = {
    "controller": "#3b82f6",
    "networking": "#f59e0b",
    "checkpoint": "#8b5cf6",
}
PRED_COLOR = "#64748b"
MEAS_COLOR = "#0ea5e9"

ALGO_HEADER_RE = re.compile(r"^#\s*algo=(\w+)", re.IGNORECASE)
ALGO_FROM_NAME_RE = re.compile(
    r"(bcrypt|sha256|sha512|md5|yescrypt)", re.IGNORECASE
)
RUN_HEADER_RE = re.compile(r"^Password\s+(.+?)\s+Worker\s+(\d+)\s*$")
METRIC_RE = re.compile(
    r"^- (.*?): count=(\d+)\s+total=([^\s]+)\s+avg=([^\s]+)\s+min=([^\s]+)\s+max=([^\s]+)(?:\s+\|\s+per-unit=([0-9]*\.?[0-9]+)ns\s+\(units=(\d+)\))?$"
)

CHART_FILES = (
    "runtime_scaling.png",
    "speedup.png",
    "overhead_breakdown.png",
    "checkpoint_impact.png",
    "prediction_vs_measured.png",
)


def duration_to_ms(value: str) -> float:
    value = value.strip().replace("μ", "µ")
    m = re.match(r"([0-9]*\.?[0-9]+)\s*(ns|µs|us|ms|s)$", value)
    if not m:
        raise ValueError(f"Unrecognized duration: {value}")

    num = float(m.group(1))
    unit = m.group(2)

    if unit == "ns":
        return num / 1_000_000.0
    if unit in ("µs", "us"):
        return num / 1000.0
    if unit == "ms":
        return num
    if unit == "s":
        return num * 1000.0

    raise ValueError(f"Unsupported unit: {unit}")


def parse_impact_percent(text: str):
    m = re.search(r"([0-9]+(?:\.[0-9]+)?)%\s+of end-to-end", text)
    return float(m.group(1)) if m else None


def infer_algorithm(path: Path, text: str) -> str:
    for line in text.splitlines()[:20]:
        m = ALGO_HEADER_RE.match(line.strip())
        if m:
            return m.group(1).lower()
    m = ALGO_FROM_NAME_RE.search(path.name)
    if m:
        return m.group(1).lower()
    return DEFAULT_ALGO


def password_sort_key(label: str) -> tuple:
    try:
        return (0, PASSWORD_ORDER.index(label))
    except ValueError:
        return (1, str(label))


def color_for(label: str) -> str:
    return PASSWORD_COLORS.get(label, "#334155")


def apply_style() -> None:
    plt.rcParams.update(
        {
            "figure.facecolor": "white",
            "axes.facecolor": "white",
            "axes.grid": True,
            "grid.alpha": 0.28,
            "grid.linestyle": "--",
            "axes.spines.top": False,
            "axes.spines.right": False,
            "font.size": 11,
            "axes.titlesize": 13,
            "axes.labelsize": 11,
            "legend.frameon": False,
            "axes.titlepad": 10,
        }
    )


def empty_summary(label: str, workers: int, algo: str) -> dict:
    return {
        "password_label": label,
        "hash_algorithm": algo,
        "number_of_workers": workers,
        "controller_side_parsing_time_ms": None,
        "job_dispatch_registration_overhead_ms": None,
        "work_assignment_overhead_total_ms": None,
        "work_assignment_overhead_per_unit_ns": None,
        "worker_cracking_time_ms": None,
        "result_return_latency_ms": None,
        "checkpoint_overhead_ms": None,
        "checkpoint_impact_pct": None,
        "total_end_to_end_runtime_ms": None,
        "controller_overhead_ms": None,
        "networking_overhead_ms": None,
        "combined_overhead_ms": None,
        "heartbeat_observation_count": None,
        "heartbeat_avg_interval_metric_ms": None,
        "checkpoint_observation_count": None,
        "checkpoint_avg_ms": None,
        "checkpoint_min_ms": None,
        "checkpoint_max_ms": None,
    }


def parse_runs(raw_text: str, hash_algorithm: str = DEFAULT_ALGO):
    """Parse `Password Ace Worker 1` blocks into summary + per-metric frames."""
    lines = [line.rstrip() for line in raw_text.splitlines()]

    summary_rows = []
    metric_rows = []

    current_label = None
    current_num_workers = None
    current_summary = None

    i = 0
    while i < len(lines):
        line = lines[i].strip()

        header_match = RUN_HEADER_RE.match(line)
        if header_match:
            if current_summary:
                summary_rows.append(current_summary)

            current_label = header_match.group(1).strip()
            current_num_workers = int(header_match.group(2))
            current_summary = empty_summary(
                current_label, current_num_workers, hash_algorithm
            )
            i += 1
            continue

        metric_match = METRIC_RE.match(line)
        if metric_match and current_label is not None:
            metric_name = metric_match.group(1).strip()
            count = int(metric_match.group(2))
            total_ms = duration_to_ms(metric_match.group(3))
            avg_ms = duration_to_ms(metric_match.group(4))
            min_ms = duration_to_ms(metric_match.group(5))
            max_ms = duration_to_ms(metric_match.group(6))
            per_unit_ns = (
                float(metric_match.group(7)) if metric_match.group(7) else None
            )
            units = int(metric_match.group(8)) if metric_match.group(8) else None

            metric_rows.append(
                {
                    "password_label": current_label,
                    "hash_algorithm": hash_algorithm,
                    "number_of_workers": current_num_workers,
                    "metric": metric_name,
                    "count": count,
                    "total_ms": total_ms,
                    "avg_ms": avg_ms,
                    "min_ms": min_ms,
                    "max_ms": max_ms,
                    "per_unit_ns": per_unit_ns,
                    "units": units,
                }
            )

            if metric_name == "controller-side parsing time":
                current_summary["controller_side_parsing_time_ms"] = total_ms
            elif metric_name == "job dispatch/registration overhead":
                current_summary["job_dispatch_registration_overhead_ms"] = total_ms
            elif metric_name == "work assignment overhead":
                current_summary["work_assignment_overhead_total_ms"] = total_ms
                current_summary["work_assignment_overhead_per_unit_ns"] = per_unit_ns
            elif metric_name == "worker cracking time (compute/search)":
                current_summary["worker_cracking_time_ms"] = total_ms
            elif metric_name == "result return latency (worker -> controller)":
                current_summary["result_return_latency_ms"] = total_ms
            elif metric_name == "checkpoint overhead observations":
                current_summary["checkpoint_overhead_ms"] = total_ms
                current_summary["checkpoint_observation_count"] = count
                current_summary["checkpoint_avg_ms"] = avg_ms
                current_summary["checkpoint_min_ms"] = min_ms
                current_summary["checkpoint_max_ms"] = max_ms
            elif metric_name == "total end-to-end runtime":
                current_summary["total_end_to_end_runtime_ms"] = total_ms

            i += 1
            continue

        if line.startswith("controller overhead:") and current_summary is not None:
            current_summary["controller_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        if line.startswith("networking overhead:") and current_summary is not None:
            current_summary["networking_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        if line.startswith("checkpoint overhead:") and current_summary is not None:
            payload = line.split(":", 1)[1].strip()
            duration_part = payload.split("(", 1)[0].strip()
            current_summary["checkpoint_overhead_ms"] = duration_to_ms(duration_part)
            current_summary["checkpoint_impact_pct"] = parse_impact_percent(payload)
            i += 1
            continue

        if line.startswith("combined overhead:") and current_summary is not None:
            current_summary["combined_overhead_ms"] = duration_to_ms(
                line.split(":", 1)[1].strip()
            )
            i += 1
            continue

        i += 1

    if current_summary:
        summary_rows.append(current_summary)

    summary_df = pd.DataFrame(summary_rows)
    metrics_df = pd.DataFrame(metric_rows)

    if not summary_df.empty:
        summary_df["run_label"] = (
            summary_df["password_label"]
            + " - "
            + summary_df["number_of_workers"].astype(str)
            + " workers"
        )

    if not metrics_df.empty:
        metrics_df["run_label"] = (
            metrics_df["password_label"]
            + " - "
            + metrics_df["number_of_workers"].astype(str)
            + " workers"
        )

    return summary_df, metrics_df


def add_prediction_columns(summary_df: pd.DataFrame) -> pd.DataFrame:
    """Predict 5-worker runtime from 1–3 worker measurements.

    Model:
        T_n = T_1 * (s + (1-s)/n)

    s is the mean of combined_overhead / total across the 1–3 worker runs.
    """
    df = summary_df.copy()
    df["serial_fraction_estimate"] = None
    df["predicted_runtime_5_workers_ms"] = None
    df["measured_runtime_5_workers_ms"] = None
    df["prediction_error_pct"] = None

    for password_label in df["password_label"].dropna().unique():
        subset = df[df["password_label"] == password_label].copy()
        subset = subset.sort_values("number_of_workers")

        one_worker = subset[subset["number_of_workers"] == 1]
        one_to_three = subset[subset["number_of_workers"].isin([1, 2, 3])]

        if one_worker.empty or one_to_three.empty:
            continue

        t1 = float(one_worker.iloc[0]["total_end_to_end_runtime_ms"])

        serial_candidates = []
        for _, row in one_to_three.iterrows():
            total = row.get("total_end_to_end_runtime_ms")
            overhead = row.get("combined_overhead_ms")
            if pd.notna(total) and pd.notna(overhead) and total > 0:
                frac = max(0.0, min(1.0, overhead / total))
                serial_candidates.append(frac)

        if not serial_candidates:
            continue

        s = sum(serial_candidates) / len(serial_candidates)
        predicted_5 = t1 * (s + (1.0 - s) / 5.0)

        df.loc[df["password_label"] == password_label, "serial_fraction_estimate"] = s
        df.loc[
            df["password_label"] == password_label, "predicted_runtime_5_workers_ms"
        ] = predicted_5

        measured_5 = subset[subset["number_of_workers"] == 5]
        if not measured_5.empty:
            measured_val = float(measured_5.iloc[0]["total_end_to_end_runtime_ms"])
            error_pct = ((measured_val - predicted_5) / predicted_5) * 100.0
            df.loc[
                df["password_label"] == password_label, "measured_runtime_5_workers_ms"
            ] = measured_val
            df.loc[df["password_label"] == password_label, "prediction_error_pct"] = (
                error_pct
            )

    return df


def add_derived_seconds(summary_df: pd.DataFrame) -> pd.DataFrame:
    df = summary_df.copy()
    df["total_end_to_end_runtime_s"] = df["total_end_to_end_runtime_ms"] / 1000.0
    df["predicted_runtime_5_workers_s"] = (
        pd.to_numeric(df["predicted_runtime_5_workers_ms"], errors="coerce") / 1000.0
    )
    df["measured_runtime_5_workers_s"] = (
        pd.to_numeric(df["measured_runtime_5_workers_ms"], errors="coerce") / 1000.0
    )

    speedup = []
    for _, row in df.iterrows():
        one = df[
            (df["password_label"] == row["password_label"])
            & (df["number_of_workers"] == 1)
        ]
        five = df[
            (df["password_label"] == row["password_label"])
            & (df["number_of_workers"] == 5)
        ]
        if one.empty or five.empty:
            speedup.append(None)
            continue
        t1 = float(one.iloc[0]["total_end_to_end_runtime_ms"])
        t5 = float(five.iloc[0]["total_end_to_end_runtime_ms"])
        speedup.append((t1 / t5) if t5 else None)

    df["speedup_1_to_5"] = speedup
    return df


def list_text_files() -> list[Path]:
    if not DATA_DIR.is_dir():
        return []

    files = []
    for path in sorted(DATA_DIR.iterdir()):
        if not path.is_file() or path.name.startswith("."):
            continue
        if path.suffix.lower() in {".png", ".csv", ".md", ".py", ".json"}:
            continue
        files.append(path)
    return files


def list_json_run_files() -> list[Path]:
    files = []
    for directory in (RUNS_DIR, DATA_DIR):
        if not directory.is_dir():
            continue
        for path in sorted(directory.glob("*.json")):
            if path.name.endswith(".controller.json"):
                continue
            files.append(path)
    return files


def summary_from_json(path: Path) -> dict:
    data = json.loads(path.read_text(encoding="utf-8"))
    if "password_label" not in data or "number_of_workers" not in data:
        raise ValueError(f"{path} is missing password_label/number_of_workers")

    row = empty_summary(
        str(data["password_label"]),
        int(data["number_of_workers"]),
        str(data.get("hash_algorithm") or infer_algorithm(path, "")),
    )
    for key in JSON_SUMMARY_FIELDS:
        if key in data and data[key] is not None:
            row[key] = data[key]
    row["run_label"] = (
        f"{row['password_label']} - {row['number_of_workers']} workers"
    )
    return row


def finalize_frames(
    summaries: list[pd.DataFrame], metrics: list[pd.DataFrame]
) -> tuple[pd.DataFrame, pd.DataFrame]:
    if not summaries:
        raise RuntimeError(
            "No runs were parsed. Run `make graphs` or add JSON under graphing/output/runs/."
        )

    summary_df = pd.concat(summaries, ignore_index=True)
    metrics_df = (
        pd.concat(metrics, ignore_index=True) if metrics else pd.DataFrame()
    )

    summary_df = summary_df.drop_duplicates(
        subset=["password_label", "number_of_workers", "hash_algorithm"],
        keep="last",
    )
    summary_df["_pw_order"] = summary_df["password_label"].map(password_sort_key)
    summary_df = summary_df.sort_values(
        ["_pw_order", "number_of_workers"]
    ).drop(columns=["_pw_order"]).reset_index(drop=True)

    if not metrics_df.empty:
        metrics_df["_pw_order"] = metrics_df["password_label"].map(password_sort_key)
        metrics_df = metrics_df.sort_values(
            ["_pw_order", "number_of_workers", "metric"]
        ).drop(columns=["_pw_order"]).reset_index(drop=True)

    return summary_df, metrics_df


def load_all_runs() -> tuple[pd.DataFrame, pd.DataFrame]:
    json_files = list_json_run_files()
    if json_files:
        print(f"Loading {len(json_files)} JSON run(s)")
        summaries = [pd.DataFrame([summary_from_json(path)]) for path in json_files]
        return finalize_frames(summaries, [])

    text_files = list_text_files()
    if not text_files:
        raise RuntimeError(
            "No JSON runs or text logs found. Run `make graphs` to execute the suite."
        )

    summaries = []
    metrics = []
    for path in text_files:
        text = path.read_text(encoding="utf-8")
        algo = infer_algorithm(path, text)
        summary_df, metrics_df = parse_runs(text, hash_algorithm=algo)
        if summary_df.empty:
            print(f"warning: no runs parsed from {path.name}")
            continue
        summaries.append(summary_df)
        if not metrics_df.empty:
            metrics.append(metrics_df)

    return finalize_frames(summaries, metrics)


def ordered_passwords(df: pd.DataFrame) -> list[str]:
    labels = list(df["password_label"].dropna().unique())
    return sorted(labels, key=password_sort_key)


def save_runtime_scaling(summary_df: pd.DataFrame, out_path: Path) -> None:
    fig, ax = plt.subplots(figsize=(9.5, 5.4))
    for label in ordered_passwords(summary_df):
        sub = summary_df[summary_df["password_label"] == label].sort_values(
            "number_of_workers"
        )
        ax.plot(
            sub["number_of_workers"],
            sub["total_end_to_end_runtime_s"],
            marker="o",
            linewidth=2.0,
            markersize=7,
            color=color_for(label),
            label=label,
        )
    ax.set_xlabel("Number of workers")
    ax.set_ylabel("End-to-end runtime (seconds)")
    ax.set_title("Runtime scaling by worker count")
    ax.set_xticks(sorted(summary_df["number_of_workers"].unique()))
    ax.legend(title="Password", loc="upper right")
    fig.tight_layout()
    fig.savefig(out_path, dpi=200)
    plt.close(fig)


def save_speedup(summary_df: pd.DataFrame, out_path: Path) -> None:
    rows = []
    for label in ordered_passwords(summary_df):
        sub = summary_df[summary_df["password_label"] == label]
        one = sub[sub["number_of_workers"] == 1]
        five = sub[sub["number_of_workers"] == 5]
        if one.empty or five.empty:
            continue
        t1 = float(one.iloc[0]["total_end_to_end_runtime_s"])
        t5 = float(five.iloc[0]["total_end_to_end_runtime_s"])
        if t5 <= 0:
            continue
        rows.append({"password_label": label, "speedup": t1 / t5})

    if not rows:
        raise RuntimeError("Need 1-worker and 5-worker runs to plot speedup.")

    df = pd.DataFrame(rows)
    fig, ax = plt.subplots(figsize=(8.4, 5.2))
    colors = [color_for(label) for label in df["password_label"]]
    bars = ax.bar(df["password_label"], df["speedup"], color=colors, width=0.62)
    ax.axhline(5.0, color="#94a3b8", linestyle="--", linewidth=1.2, label="Ideal 5×")
    ax.set_xlabel("Password")
    ax.set_ylabel("Speedup (1-worker / 5-worker)")
    ax.set_title("Speedup at 5 workers vs 1 worker")
    for bar, value in zip(bars, df["speedup"]):
        ax.text(
            bar.get_x() + bar.get_width() / 2,
            bar.get_height() + 0.08,
            f"{value:.2f}×",
            ha="center",
            va="bottom",
            fontsize=10,
        )
    ax.set_ylim(0, max(df["speedup"].max() * 1.18, 5.6))
    ax.legend(loc="upper right")
    fig.tight_layout()
    fig.savefig(out_path, dpi=200)
    plt.close(fig)


def save_overhead_breakdown(summary_df: pd.DataFrame, out_path: Path) -> None:
    fig, axes = plt.subplots(1, 2, figsize=(11.2, 5.4), sharey=True)
    labels = ordered_passwords(summary_df)
    x = range(len(labels))

    for ax, workers, title in (
        (axes[0], 1, "1 worker"),
        (axes[1], 5, "5 workers"),
    ):
        subset = summary_df[summary_df["number_of_workers"] == workers]
        controller = []
        networking = []
        checkpoint = []
        for label in labels:
            row = subset[subset["password_label"] == label]
            if row.empty:
                controller.append(0.0)
                networking.append(0.0)
                checkpoint.append(0.0)
            else:
                controller.append(
                    float(row.iloc[0]["controller_overhead_ms"] or 0) / 1000.0
                )
                networking.append(
                    float(row.iloc[0]["networking_overhead_ms"] or 0) / 1000.0
                )
                checkpoint.append(
                    float(row.iloc[0]["checkpoint_overhead_ms"] or 0) / 1000.0
                )

        ax.bar(
            x,
            controller,
            color=OVERHEAD_COLORS["controller"],
            label="Controller",
            width=0.64,
        )
        ax.bar(
            x,
            networking,
            bottom=controller,
            color=OVERHEAD_COLORS["networking"],
            label="Networking",
            width=0.64,
        )
        ax.bar(
            x,
            checkpoint,
            bottom=[c + n for c, n in zip(controller, networking)],
            color=OVERHEAD_COLORS["checkpoint"],
            label="Checkpoint",
            width=0.64,
        )
        ax.set_title(title)
        ax.set_xticks(list(x))
        ax.set_xticklabels(labels)
        ax.set_xlabel("Password")

    axes[0].set_ylabel("Overhead (seconds)")
    fig.suptitle("Overhead breakdown: controller, networking, checkpoint")
    handles, legend_labels = axes[0].get_legend_handles_labels()
    fig.legend(
        handles,
        legend_labels,
        loc="lower center",
        ncol=3,
        bbox_to_anchor=(0.5, -0.04),
    )
    fig.tight_layout()
    fig.savefig(out_path, dpi=200, bbox_inches="tight")
    plt.close(fig)


def save_checkpoint_impact(summary_df: pd.DataFrame, out_path: Path) -> None:
    fig, ax = plt.subplots(figsize=(9.5, 5.4))
    for label in ordered_passwords(summary_df):
        sub = summary_df[summary_df["password_label"] == label].sort_values(
            "number_of_workers"
        )
        ax.plot(
            sub["number_of_workers"],
            sub["checkpoint_impact_pct"],
            marker="o",
            linewidth=2.0,
            markersize=7,
            color=color_for(label),
            label=label,
        )
    ax.set_xlabel("Number of workers")
    ax.set_ylabel("Checkpoint overhead (% of end-to-end runtime)")
    ax.set_title("Checkpoint impact vs worker count")
    ax.set_xticks(sorted(summary_df["number_of_workers"].unique()))
    ax.legend(title="Password", loc="upper left")
    fig.tight_layout()
    fig.savefig(out_path, dpi=200)
    plt.close(fig)


def save_prediction_vs_measured(summary_df: pd.DataFrame, out_path: Path) -> None:
    rows = []
    for label in ordered_passwords(summary_df):
        subset = summary_df[summary_df["password_label"] == label]
        pred = subset["predicted_runtime_5_workers_s"].dropna()
        meas = subset["measured_runtime_5_workers_s"].dropna()
        if pred.empty or meas.empty:
            continue
        rows.append(
            {
                "password_label": label,
                "predicted": float(pred.iloc[0]),
                "measured": float(meas.iloc[0]),
            }
        )

    if not rows:
        raise RuntimeError("Need Amdahl predictions and 5-worker measurements.")

    df = pd.DataFrame(rows)
    x = range(len(df))
    width = 0.36

    fig, ax = plt.subplots(figsize=(9.2, 5.4))
    ax.bar(
        [i - width / 2 for i in x],
        df["predicted"],
        width=width,
        color=PRED_COLOR,
        label="Amdahl prediction (5 workers)",
    )
    ax.bar(
        [i + width / 2 for i in x],
        df["measured"],
        width=width,
        color=MEAS_COLOR,
        label="Measured (5 workers)",
    )
    ax.set_xticks(list(x))
    ax.set_xticklabels(df["password_label"])
    ax.set_xlabel("Password")
    ax.set_ylabel("5-worker runtime (seconds)")
    ax.set_title("Amdahl 5-worker prediction vs measured runtime")
    ax.legend(loc="upper left")
    fig.tight_layout()
    fig.savefig(out_path, dpi=200)
    plt.close(fig)


def one_vs_five_rows(summary_df: pd.DataFrame) -> list[dict]:
    rows = []
    for label in ordered_passwords(summary_df):
        sub = summary_df[summary_df["password_label"] == label]
        one = sub[sub["number_of_workers"] == 1]
        five = sub[sub["number_of_workers"] == 5]
        if one.empty or five.empty:
            continue
        t1 = float(one.iloc[0]["total_end_to_end_runtime_s"])
        t5 = float(five.iloc[0]["total_end_to_end_runtime_s"])
        rows.append(
            {
                "label": label,
                "t1": t1,
                "t5": t5,
                "speedup": (t1 / t5) if t5 else float("nan"),
                "prediction_error_pct": float(five.iloc[0]["prediction_error_pct"]),
                "checkpoint_impact_pct": float(five.iloc[0]["checkpoint_impact_pct"]),
            }
        )
    return rows


def print_one_vs_five(summary_df: pd.DataFrame) -> None:
    print("\n1-worker vs 5-worker runtime (seconds)")
    print(
        f"{'Password':<10} {'1 worker (s)':>14} {'5 workers (s)':>15} {'Speedup':>10}"
    )
    for row in one_vs_five_rows(summary_df):
        print(
            f"{row['label']:<10} {row['t1']:14.3f} {row['t5']:15.3f} {row['speedup']:10.2f}×"
        )


def _signed_pct_range(values: list[float]) -> str:
    lo, hi = min(values), max(values)

    def fmt(value: float) -> str:
        sign = "+" if value >= 0 else ""
        return f"{sign}{round(value)}%"

    return f"{fmt(lo)} to {fmt(hi)}"


def readme_benchmark_table(rows: list[dict]) -> str:
    lines = [
        "| Password | 1 worker | 5 workers | Speedup |",
        "| --- | ---: | ---: | ---: |",
    ]
    for row in rows:
        lines.append(
            f"| {row['label']} | {row['t1']:.2f}s | {row['t5']:.2f}s | {row['speedup']:.2f}x |"
        )
    return "\n".join(lines) + "\n"


def readme_speedup_sentence(rows: list[dict]) -> str:
    speedups = [row["speedup"] for row in rows]
    errors = [row["prediction_error_pct"] for row in rows]
    checkpoints = [row["checkpoint_impact_pct"] for row in rows]
    return (
        f"Speedup ranges from **{min(speedups):.2f}x to {max(speedups):.2f}x** "
        f"(average **~{sum(speedups) / len(speedups):.2f}x**). "
        "An Amdahl prediction from the 1–3 worker serial fraction is optimistic here "
        f"(**{_signed_pct_range(errors)}**). "
        f"Checkpoint time at 5 workers is **{min(checkpoints):.2f}%–{max(checkpoints):.2f}%** "
        "of wall clock. Numbers come from `graphing/output/assignment_summary.csv`."
    )


def readme_prediction_sentence(rows: list[dict]) -> str:
    errors = [row["prediction_error_pct"] for row in rows]
    return (
        "A serial-fraction prediction from the 1–3 worker runs underestimates "
        "5-worker time on this machine "
        f"(**{_signed_pct_range(errors)}**)."
    )


README_TABLE_RE = re.compile(
    r"\| Password \| 1 worker \| 5 workers \| Speedup \|\n"
    r"\| --- \| ---: \| ---: \| ---: \|\n"
    r"(?:\|[^\n]+\n)+",
)
README_STATS_RE = re.compile(
    r"Speedup ranges from \*\*[^*]+\*\* \(average \*\*~[^*]+\*\*\)\. "
    r"An Amdahl prediction from the 1–3 worker serial fraction is optimistic here "
    r"\(\*\*[^*]+\*\*\)\. "
    r"Checkpoint time at 5 workers is \*\*[^*]+\*\* of wall clock\. "
    r"Numbers come from `graphing/output/assignment_summary\.csv`\."
)
README_PRED_RE = re.compile(
    r"A serial-fraction prediction from the 1–3 worker runs underestimates "
    r"5-worker time on this machine \(\*\*[^*]+\*\*\)\."
)


def update_readme(summary_df: pd.DataFrame, path: Path = README_PATH) -> None:
    rows = one_vs_five_rows(summary_df)
    if not rows:
        raise RuntimeError("Need 1-worker and 5-worker runs to update README.md.")

    text = path.read_text(encoding="utf-8")
    replacements = (
        (README_TABLE_RE, readme_benchmark_table(rows)),
        (README_STATS_RE, readme_speedup_sentence(rows)),
        (README_PRED_RE, readme_prediction_sentence(rows)),
    )
    for pattern, replacement in replacements:
        updated, count = pattern.subn(replacement, text, count=1)
        if count != 1:
            raise RuntimeError(f"Could not find README benchmark block matching {pattern.pattern}")
        text = updated

    path.write_text(text, encoding="utf-8")
    print(f"Updated: {path}")


def main() -> None:
    apply_style()
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    summary_df, metrics_df = load_all_runs()
    summary_df = add_prediction_columns(summary_df)
    summary_df = add_derived_seconds(summary_df)

    summary_path = OUT_DIR / "assignment_summary.csv"
    details_path = OUT_DIR / "assignment_metric_details.csv"
    summary_df.to_csv(summary_path, index=False)
    if not metrics_df.empty:
        metrics_df.to_csv(details_path, index=False)
    elif details_path.exists():
        details_path.unlink()

    save_runtime_scaling(summary_df, OUT_DIR / "runtime_scaling.png")
    save_speedup(summary_df, OUT_DIR / "speedup.png")
    save_overhead_breakdown(summary_df, OUT_DIR / "overhead_breakdown.png")
    save_checkpoint_impact(summary_df, OUT_DIR / "checkpoint_impact.png")
    save_prediction_vs_measured(summary_df, OUT_DIR / "prediction_vs_measured.png")

    source = RUNS_DIR if list_json_run_files() else DATA_DIR
    print(f"Parsed {len(summary_df)} runs from {source}")
    print(f"Saved: {summary_path}")
    if details_path.exists():
        print(f"Saved: {details_path}")
    for name in CHART_FILES:
        print(f"Saved: {OUT_DIR / name}")

    print_one_vs_five(summary_df)
    update_readme(summary_df)


if __name__ == "__main__":
    main()

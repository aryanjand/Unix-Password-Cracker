import re
from pathlib import Path

import pandas as pd
import matplotlib.pyplot as plt


RAW_TEXT = r"""
Password Ace Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=32.729µs avg=32.729µs min=32.729µs max=32.729µs
- job dispatch/registration overhead: count=9 total=133.293429ms avg=14.810381ms min=11.07984ms max=17.954473ms
- work assignment overhead: count=9 total=48.242597ms avg=5.360288ms min=546.999µs max=8.47388ms | per-unit=5360.29ns (units=9000)
- worker cracking time (compute/search): count=9 total=11.398773784s avg=1.26653042s min=736.415411ms max=1.382311679s
- result return latency (worker -> controller): count=1 total=317.582µs avg=317.582µs min=317.582µs max=317.582µs
- checkpoint overhead observations: count=85 total=493.275996ms avg=5.803247ms min=4.97496ms max=17.706318ms
- total end-to-end runtime: count=1 total=15.984453238s avg=15.984453238s min=15.984453238s max=15.984453238s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 48.275326ms
  networking overhead: 133.611011ms
  checkpoint overhead: 493.275996ms (moderate impact, 3.09% of end-to-end)
  combined overhead:   675.162333ms





Password Ace Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=547.973µs avg=547.973µs min=547.973µs max=547.973µs
- job dispatch/registration overhead: count=9 total=145.981998ms avg=16.220222ms min=9.854092ms max=30.45606ms
- work assignment overhead: count=8 total=35.323028ms avg=4.415378ms min=542.167µs max=5.970135ms | per-unit=4415.38ns (units=8000)
- worker cracking time (compute/search): count=8 total=9.869248023s avg=1.233656002s min=759.347767ms max=1.319021158s
- result return latency (worker -> controller): count=1 total=333.367µs avg=333.367µs min=333.367µs max=333.367µs
- checkpoint overhead observations: count=82 total=499.801391ms avg=6.095138ms min=5.044744ms max=18.005221ms
- total end-to-end runtime: count=1 total=6.340247632s avg=6.340247632s min=6.340247632s max=6.340247632s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 35.871001ms
  networking overhead: 146.315365ms
  checkpoint overhead: 499.801391ms (high impact, 7.88% of end-to-end)
  combined overhead:   681.987757ms






Password Ace Worker 3
  ===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=985.459µs avg=985.459µs min=985.459µs max=985.459µs
- job dispatch/registration overhead: count=10 total=150.511133ms avg=15.051113ms min=9.952493ms max=20.37155ms
- work assignment overhead: count=8 total=33.677799ms avg=4.209724ms min=597.678µs max=7.777268ms | per-unit=4209.72ns (units=8000)
- worker cracking time (compute/search): count=8 total=9.911205317s avg=1.238900664s min=734.107991ms max=1.335132216s
- result return latency (worker -> controller): count=1 total=285.569µs avg=285.569µs min=285.569µs max=285.569µs
- checkpoint overhead observations: count=84 total=529.268695ms avg=6.300817ms min=4.954392ms max=18.488848ms
- total end-to-end runtime: count=1 total=13.645110187s avg=13.645110187s min=13.645110187s max=13.645110187s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 34.663258ms
  networking overhead: 150.796702ms
  checkpoint overhead: 529.268695ms (moderate impact, 3.88% of end-to-end)
  combined overhead:   714.728655ms






Password Bad Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=598.565µs avg=598.565µs min=598.565µs max=598.565µs
- job dispatch/registration overhead: count=15 total=214.532446ms avg=14.302163ms min=9.293679ms max=21.637092ms
- work assignment overhead: count=15 total=81.795756ms avg=5.45305ms min=685.078µs max=9.988845ms | per-unit=5453.05ns (units=15000)
- worker cracking time (compute/search): count=15 total=19.347603472s avg=1.289840231s min=837.69743ms max=1.368393691s
- result return latency (worker -> controller): count=1 total=360.437µs avg=360.437µs min=360.437µs max=360.437µs
- checkpoint overhead observations: count=146 total=861.781524ms avg=5.902613ms min=2.841464ms max=22.586794ms
- total end-to-end runtime: count=1 total=22.122574053s avg=22.122574053s min=22.122574053s max=22.122574053s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 82.394321ms
  networking overhead: 214.892883ms
  checkpoint overhead: 861.781524ms (moderate impact, 3.90% of end-to-end)
  combined overhead:   1.159068728s






Password Bad Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=511.905µs avg=511.905µs min=511.905µs max=511.905µs
- job dispatch/registration overhead: count=16 total=246.146835ms avg=15.384177ms min=10.790674ms max=23.517101ms
- work assignment overhead: count=15 total=78.68278ms avg=5.245518ms min=433.269µs max=8.005987ms | per-unit=5245.52ns (units=15000)
- worker cracking time (compute/search): count=15 total=19.509856707s avg=1.300657113s min=887.614591ms max=1.383137393s
- result return latency (worker -> controller): count=1 total=430.681µs avg=430.681µs min=430.681µs max=430.681µs
- checkpoint overhead observations: count=151 total=942.955194ms avg=6.244736ms min=5.009305ms max=24.770662ms
- total end-to-end runtime: count=1 total=13.742855529s avg=13.742855529s min=13.742855529s max=13.742855529s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 79.194685ms
  networking overhead: 246.577516ms
  checkpoint overhead: 942.955194ms (high impact, 6.86% of end-to-end)
  combined overhead:   1.268727395s







Password Bad Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=2.967175ms avg=2.967175ms min=2.967175ms max=2.967175ms
- job dispatch/registration overhead: count=15 total=230.529303ms avg=15.36862ms min=10.794735ms max=24.210989ms
- work assignment overhead: count=13 total=68.352382ms avg=5.257875ms min=603.233µs max=13.970061ms | per-unit=5257.88ns (units=13000)
- worker cracking time (compute/search): count=13 total=16.658127372s avg=1.281394413s min=885.923726ms max=1.383429594s
- result return latency (worker -> controller): count=1 total=364.935µs avg=364.935µs min=364.935µs max=364.935µs
- checkpoint overhead observations: count=141 total=886.76814ms avg=6.289135ms min=4.947735ms max=20.443034ms
- total end-to-end runtime: count=1 total=18.113060535s avg=18.113060535s min=18.113060535s max=18.113060535s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 71.319557ms
  networking overhead: 230.894238ms
  checkpoint overhead: 886.76814ms (moderate impact, 4.90% of end-to-end)
  combined overhead:   1.188981935s







Password Cab Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=616.361µs avg=616.361µs min=616.361µs max=616.361µs
- job dispatch/registration overhead: count=21 total=366.700523ms avg=17.461929ms min=13.405513ms max=60.881557ms
- work assignment overhead: count=21 total=160.549854ms avg=7.645231ms min=447.333µs max=54.889822ms | per-unit=7645.23ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.368996964s avg=1.303285569s min=1.166429589s max=1.360529507s
- result return latency (worker -> controller): count=1 total=343.472µs avg=343.472µs min=343.472µs max=343.472µs
- checkpoint overhead observations: count=208 total=1.285117262s avg=6.178448ms min=4.836577ms max=66.857124ms
- total end-to-end runtime: count=1 total=29.739544795s avg=29.739544795s min=29.739544795s max=29.739544795s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 161.166215ms
  networking overhead: 367.043995ms
  checkpoint overhead: 1.285117262s (moderate impact, 4.32% of end-to-end)
  combined overhead:   1.813327472s







Password Cab Worker 2
- controller-side parsing time: count=1 total=2.393922ms avg=2.393922ms min=2.393922ms max=2.393922ms
- job dispatch/registration overhead: count=22 total=345.615585ms avg=15.709799ms min=10.684107ms max=33.613582ms
- work assignment overhead: count=21 total=111.23026ms avg=5.296679ms min=448.187µs max=8.907313ms | per-unit=5296.68ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.292197581s avg=1.299628456s min=1.137404649s max=1.345951933s
- result return latency (worker -> controller): count=1 total=306.741µs avg=306.741µs min=306.741µs max=306.741µs
- checkpoint overhead observations: count=215 total=1.341818618s avg=6.241016ms min=4.845168ms max=34.203622ms
- total end-to-end runtime: count=1 total=16.064055635s avg=16.064055635s min=16.064055635s max=16.064055635s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 113.624182ms
  networking overhead: 345.922326ms
  checkpoint overhead: 1.341818618s (high impact, 8.35% of end-to-end)
  combined overhead:   1.801365126s







Password Cab Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=572.76µs avg=572.76µs min=572.76µs max=572.76µs
- job dispatch/registration overhead: count=23 total=340.985932ms avg=14.825475ms min=10.0644ms max=17.646453ms
- work assignment overhead: count=21 total=102.048954ms avg=4.859474ms min=399.258µs max=6.271847ms | per-unit=4859.47ns (units=21000)
- worker cracking time (compute/search): count=21 total=27.402232377s avg=1.304868208s min=1.147081996s max=1.369105088s
- result return latency (worker -> controller): count=1 total=407.608µs avg=407.608µs min=407.608µs max=407.608µs
- checkpoint overhead observations: count=213 total=1.330036987s avg=6.244305ms min=4.971795ms max=19.419495ms
- total end-to-end runtime: count=1 total=12.037563992s avg=12.037563992s min=12.037563992s max=12.037563992s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 102.621714ms
  networking overhead: 341.39354ms
  checkpoint overhead: 1.330036987s (high impact, 11.05% of end-to-end)
  combined overhead:   1.774052241s







Password Dad Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=554.102µs avg=554.102µs min=554.102µs max=554.102µs
- job dispatch/registration overhead: count=28 total=429.466219ms avg=15.338079ms min=11.672591ms max=18.669758ms
- work assignment overhead: count=28 total=153.887353ms avg=5.495976ms min=495.235µs max=8.216611ms | per-unit=5495.98ns (units=28000)
- worker cracking time (compute/search): count=28 total=35.269688018s avg=1.259631714s min=163.334022ms max=1.395953669s
- result return latency (worker -> controller): count=1 total=334.952µs avg=334.952µs min=334.952µs max=334.952µs
- checkpoint overhead observations: count=271 total=1.735327857s avg=6.403423ms min=5.030963ms max=26.688831ms
- total end-to-end runtime: count=1 total=37.35632461s avg=37.35632461s min=37.35632461s max=37.35632461s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 154.441455ms
  networking overhead: 429.801171ms
  checkpoint overhead: 1.735327857s (moderate impact, 4.65% of end-to-end)
  combined overhead:   2.319570483s








Password Dad Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=2.393118ms avg=2.393118ms min=2.393118ms max=2.393118ms
- job dispatch/registration overhead: count=28 total=416.115963ms avg=14.861284ms min=10.678874ms max=18.091372ms
- work assignment overhead: count=27 total=142.548273ms avg=5.279565ms min=304.932µs max=7.130386ms | per-unit=5279.57ns (units=27000)
- worker cracking time (compute/search): count=27 total=34.309117095s avg=1.27070804s min=165.172894ms max=1.348590236s
- result return latency (worker -> controller): count=1 total=406.112µs avg=406.112µs min=406.112µs max=406.112µs
- checkpoint overhead observations: count=270 total=1.664555894s avg=6.165021ms min=4.936946ms max=23.81527ms
- total end-to-end runtime: count=1 total=18.767757168s avg=18.767757168s min=18.767757168s max=18.767757168s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 144.941391ms
  networking overhead: 416.522075ms
  checkpoint overhead: 1.664555894s (high impact, 8.87% of end-to-end)
  combined overhead:   2.22601936s







Password Dad Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=631.081µs avg=631.081µs min=631.081µs max=631.081µs
- job dispatch/registration overhead: count=30 total=451.9247ms avg=15.064156ms min=10.669615ms max=20.209473ms
- work assignment overhead: count=28 total=148.675429ms avg=5.309836ms min=453.707µs max=7.734798ms | per-unit=5309.84ns (units=28000)
- worker cracking time (compute/search): count=28 total=35.878326965s avg=1.28136882s min=168.0746ms max=1.404559625s
- result return latency (worker -> controller): count=1 total=310.807µs avg=310.807µs min=310.807µs max=310.807µs
- checkpoint overhead observations: count=271 total=1.722302159s avg=6.355358ms min=5.028086ms max=20.354057ms
- total end-to-end runtime: count=1 total=14.061031919s avg=14.061031919s min=14.061031919s max=14.061031919s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 149.30651ms
  networking overhead: 452.235507ms
  checkpoint overhead: 1.722302159s (high impact, 12.25% of end-to-end)
  combined overhead:   2.323844176s








Password Ear Worker 1
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=495.019µs avg=495.019µs min=495.019µs max=495.019µs
- job dispatch/registration overhead: count=34 total=549.788668ms avg=16.170254ms min=14.294737ms max=25.770127ms
- work assignment overhead: count=34 total=187.402131ms avg=5.511827ms min=341.085µs max=7.562308ms | per-unit=5511.83ns (units=34000)
- worker cracking time (compute/search): count=34 total=43.353517731s avg=1.275103462s min=496.926035ms max=1.340598088s
- result return latency (worker -> controller): count=1 total=361.48µs avg=361.48µs min=361.48µs max=361.48µs
- checkpoint overhead observations: count=333 total=2.03962499s avg=6.124999ms min=4.861944ms max=26.345092ms
- total end-to-end runtime: count=1 total=43.990097333s avg=43.990097333s min=43.990097333s max=43.990097333s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 187.89715ms
  networking overhead: 550.150148ms
  checkpoint overhead: 2.03962499s (moderate impact, 4.64% of end-to-end)
  combined overhead:   2.777672288s







Password Ear Worker 2
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=558.155µs avg=558.155µs min=558.155µs max=558.155µs
- job dispatch/registration overhead: count=34 total=555.42052ms avg=16.335897ms min=9.98262ms max=59.721239ms
- work assignment overhead: count=33 total=206.150565ms avg=6.246986ms min=698.915µs max=33.492776ms | per-unit=6246.99ns (units=33000)
- worker cracking time (compute/search): count=33 total=42.77220796s avg=1.296127513s min=496.359518ms max=1.454176549s
- result return latency (worker -> controller): count=1 total=321.904µs avg=321.904µs min=321.904µs max=321.904µs
- checkpoint overhead observations: count=332 total=2.169429306s avg=6.534425ms min=4.939216ms max=46.779899ms
- total end-to-end runtime: count=1 total=23.699949608s avg=23.699949608s min=23.699949608s max=23.699949608s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 206.70872ms
  networking overhead: 555.742424ms
  checkpoint overhead: 2.169429306s (high impact, 9.15% of end-to-end)
  combined overhead:   2.93188045s










Password Ear Worker 3
===== Runtime Metrics Summary =====
- controller-side parsing time: count=1 total=575.37µs avg=575.37µs min=575.37µs max=575.37µs
- job dispatch/registration overhead: count=36 total=537.190711ms avg=14.921964ms min=9.93148ms max=20.908478ms
- work assignment overhead: count=34 total=181.358218ms avg=5.334065ms min=324.833µs max=11.026071ms | per-unit=5334.07ns (units=34000)
- worker cracking time (compute/search): count=34 total=44.008027102s avg=1.294353738s min=500.587501ms max=1.382185416s
- result return latency (worker -> controller): count=1 total=530.841µs avg=530.841µs min=530.841µs max=530.841µs
- checkpoint overhead observations: count=334 total=1.973302173s avg=5.90809ms min=4.964412ms max=20.301883ms
- total end-to-end runtime: count=1 total=16.392842194s avg=16.392842194s min=16.392842194s max=16.392842194s

Main results (controller + networking + checkpoint overhead)
  controller overhead: 181.933588ms
  networking overhead: 537.721552ms
  checkpoint overhead: 1.973302173s (high impact, 12.04% of end-to-end)
  combined overhead:   2.692957313s
"""


def duration_to_ms(value: str) -> float:
    """
    Convert duration strings like:
    32.729µs, 547.973us, 133.293429ms, 11.398773784s, 2.967175ms
    into milliseconds.
    """
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


def parse_runs(raw_text: str):
    """
    Returns:
      summary_df: one row per Password / Number of Workers
      metrics_df: one row per metric per Password / Number of Workers
    """
    lines = [line.rstrip() for line in raw_text.splitlines()]

    # Example:
    # Password Ace Worker 1
    # -> password = Ace
    # -> number_of_workers = 1
    run_header_re = re.compile(r"^Password\s+(.+?)\s+Worker\s+(\d+)\s*$")

    metric_re = re.compile(
        r"^- (.*?): count=(\d+)\s+total=([^\s]+)\s+avg=([^\s]+)\s+min=([^\s]+)\s+max=([^\s]+)(?:\s+\|\s+per-unit=([0-9]*\.?[0-9]+)ns\s+\(units=(\d+)\))?$"
    )

    summary_rows = []
    metric_rows = []

    current_password = None
    current_num_workers = None
    current_summary = None

    i = 0
    while i < len(lines):
        line = lines[i].strip()

        header_match = run_header_re.match(line)
        if header_match:
            if current_summary:
                summary_rows.append(current_summary)

            current_password = header_match.group(1).strip()
            current_num_workers = int(header_match.group(2))

            current_summary = {
                "password": current_password,
                "number_of_workers": current_num_workers,
                "controller_overhead_ms": None,
                "networking_overhead_ms": None,
                "checkpoint_overhead_ms": None,
                "checkpoint_impact_pct": None,
                "combined_overhead_ms": None,
                "end_to_end_runtime_ms": None,
            }
            i += 1
            continue

        metric_match = metric_re.match(line)
        if metric_match and current_password is not None:
            metric_name = metric_match.group(1).strip()
            count = int(metric_match.group(2))
            total_ms = duration_to_ms(metric_match.group(3))
            avg_ms = duration_to_ms(metric_match.group(4))
            min_ms = duration_to_ms(metric_match.group(5))
            max_ms = duration_to_ms(metric_match.group(6))
            per_unit_ns = float(metric_match.group(7)) if metric_match.group(7) else None
            units = int(metric_match.group(8)) if metric_match.group(8) else None

            metric_rows.append(
                {
                    "password": current_password,
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

            if metric_name == "total end-to-end runtime":
                current_summary["end_to_end_runtime_ms"] = total_ms

            i += 1
            continue

        if line.startswith("controller overhead:") and current_summary is not None:
            current_summary["controller_overhead_ms"] = duration_to_ms(line.split(":", 1)[1].strip())
            i += 1
            continue

        if line.startswith("networking overhead:") and current_summary is not None:
            current_summary["networking_overhead_ms"] = duration_to_ms(line.split(":", 1)[1].strip())
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
            current_summary["combined_overhead_ms"] = duration_to_ms(line.split(":", 1)[1].strip())
            i += 1
            continue

        i += 1

    if current_summary:
        summary_rows.append(current_summary)

    summary_df = pd.DataFrame(summary_rows)
    metrics_df = pd.DataFrame(metric_rows)

    summary_df["run_label"] = (
        summary_df["password"] + " - " + summary_df["number_of_workers"].astype(str) + " workers"
    )
    metrics_df["run_label"] = (
        metrics_df["password"] + " - " + metrics_df["number_of_workers"].astype(str) + " workers"
    )

    return summary_df, metrics_df


def save_summary_table_image(df: pd.DataFrame, output_path: str):
    table_df = df[
        [
            "password",
            "number_of_workers",
            "end_to_end_runtime_ms",
            "controller_overhead_ms",
            "networking_overhead_ms",
            "checkpoint_overhead_ms",
            "combined_overhead_ms",
            "checkpoint_impact_pct",
        ]
    ].copy()

    numeric_cols = [
        "end_to_end_runtime_ms",
        "controller_overhead_ms",
        "networking_overhead_ms",
        "checkpoint_overhead_ms",
        "combined_overhead_ms",
        "checkpoint_impact_pct",
    ]
    for col in numeric_cols:
        table_df[col] = table_df[col].round(3)

    fig_height = max(4, 0.4 * len(table_df) + 1.5)
    fig, ax = plt.subplots(figsize=(14, fig_height))
    ax.axis("off")

    tbl = ax.table(
        cellText=table_df.values,
        colLabels=table_df.columns,
        loc="center",
        cellLoc="center",
    )
    tbl.auto_set_font_size(False)
    tbl.set_fontsize(8)
    tbl.scale(1, 1.3)

    plt.tight_layout()
    plt.savefig(output_path, dpi=200, bbox_inches="tight")
    plt.close()


def create_charts(summary_df: pd.DataFrame, out_dir: Path):
    plot_df = summary_df.sort_values(["password", "number_of_workers"]).copy()

    # 1) End-to-end runtime bar chart
    plt.figure(figsize=(14, 6))
    plt.bar(plot_df["run_label"], plot_df["end_to_end_runtime_ms"])
    plt.xticks(rotation=60, ha="right")
    plt.ylabel("Runtime (ms)")
    plt.title("End-to-End Runtime by Password / Number of Workers")
    plt.tight_layout()
    plt.savefig(out_dir / "end_to_end_runtime_bar.png", dpi=200)
    plt.close()

    # 2) Overhead breakdown stacked bar chart
    plt.figure(figsize=(14, 6))
    x = range(len(plot_df))
    c = plot_df["controller_overhead_ms"].fillna(0)
    n = plot_df["networking_overhead_ms"].fillna(0)
    ck = plot_df["checkpoint_overhead_ms"].fillna(0)

    plt.bar(x, c, label="Controller")
    plt.bar(x, n, bottom=c, label="Networking")
    plt.bar(x, ck, bottom=c + n, label="Checkpoint")

    plt.xticks(list(x), plot_df["run_label"], rotation=60, ha="right")
    plt.ylabel("Overhead (ms)")
    plt.title("Overhead Breakdown by Password / Number of Workers")
    plt.legend()
    plt.tight_layout()
    plt.savefig(out_dir / "overhead_breakdown_stacked_bar.png", dpi=200)
    plt.close()

    # 3) Average runtime by password
    avg_df = (
        plot_df.groupby("password", as_index=False)["end_to_end_runtime_ms"]
        .mean()
        .sort_values("end_to_end_runtime_ms")
    )

    plt.figure(figsize=(10, 5))
    plt.bar(avg_df["password"], avg_df["end_to_end_runtime_ms"])
    plt.ylabel("Average Runtime (ms)")
    plt.title("Average End-to-End Runtime by Password")
    plt.tight_layout()
    plt.savefig(out_dir / "avg_runtime_by_password_bar.png", dpi=200)
    plt.close()


def main():
    out_dir = Path("normalized_output")
    out_dir.mkdir(exist_ok=True)

    summary_df, metrics_df = parse_runs(RAW_TEXT)

    if summary_df.empty:
        raise RuntimeError("No runs were parsed. Check RAW_TEXT formatting.")

    summary_df = summary_df.sort_values(["password", "number_of_workers"]).reset_index(drop=True)
    metrics_df = metrics_df.sort_values(["password", "number_of_workers", "metric"]).reset_index(drop=True)

    summary_csv = out_dir / "run_summary.csv"
    metrics_csv = out_dir / "metric_details.csv"

    summary_df.to_csv(summary_csv, index=False)
    metrics_df.to_csv(metrics_csv, index=False)

    create_charts(summary_df, out_dir)
    save_summary_table_image(summary_df, out_dir / "run_summary_table.png")

    print(f"Saved: {summary_csv}")
    print(f"Saved: {metrics_csv}")
    print(f"Saved: {out_dir / 'end_to_end_runtime_bar.png'}")
    print(f"Saved: {out_dir / 'overhead_breakdown_stacked_bar.png'}")
    print(f"Saved: {out_dir / 'avg_runtime_by_password_bar.png'}")
    print(f"Saved: {out_dir / 'run_summary_table.png'}")

    print("\nPreview: run_summary.csv")
    print(summary_df.head(10).to_string(index=False))

    print("\nPreview: metric_details.csv")
    print(metrics_df.head(10).to_string(index=False))


if __name__ == "__main__":
    main()
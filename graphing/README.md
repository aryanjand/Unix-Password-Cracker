# Graphing

`make graphs` is the full pipeline: build the current Go binaries, run the bcrypt fixture suite, write JSON metrics, then draw the README charts.

```bash
make graphs
```

`make plots` redraws from the last `graphing/output/runs/*.json` (or the older text logs in `graphing/data/`) without starting workers.

## What the suite does

For Ace, Bad, Cab, Dad, and Ear it starts one controller and N workers (`N` = 1, 2, 3, 5), using `testdata/shadow/shadow_*_bcrypt`. The controller writes millisecond metrics with `-metrics-json`. `run_suite.py` only adds the labels (password, worker count, algorithm).

Subset while iterating:

```bash
./graphing/generate.sh --passwords Ace --workers 1
```

## Output

- `graphing/output/runs/<password>_w<n>.json` — one structured run
- `graphing/output/*.png` and `assignment_summary.csv` — README artifacts
- root `README.md` benchmark table and the two number sentences next to it

# Graphing

One-command charts from pasted controller metric logs.

```bash
make graphs
```

That creates `graphing/.venv` if needed, installs `graphing/requirements.txt`, and writes CSVs plus PNGs to `graphing/output/`.

## Adding a new run

1. Copy the controller's `===== Runtime Metrics Summary =====` block.
2. Put a paste-time label on the line above it. The Go controller does not emit this line:

   ```
   Password Ace Worker 1
   ```

   Use the password nickname and the worker count from that run.

3. Optional header at the top of the file (otherwise the filename is checked for `bcrypt` / `sha256` / `sha512` / `md5` / `yescrypt`, then it defaults to bcrypt):

   ```
   # algo=bcrypt
   ```

4. Drop the file in `graphing/data/` (any text file; comments starting with `#` are ignored).
5. Run `make graphs` again from the repo root (or `./generate.sh` from this directory).

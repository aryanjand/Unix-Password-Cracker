#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${SCRIPT_DIR}/.venv"
REQ_FILE="${SCRIPT_DIR}/requirements.txt"
SUITE_PY="${SCRIPT_DIR}/run_suite.py"
GEN_PY="${SCRIPT_DIR}/generate.py"

PLOT_ONLY=0
SUITE_ARGS=()
for arg in "$@"; do
  if [[ "${arg}" == "--plot-only" ]]; then
    PLOT_ONLY=1
  else
    SUITE_ARGS+=("${arg}")
  fi
done

if [[ ! -d "${VENV_DIR}" ]]; then
  echo "Creating virtualenv at ${VENV_DIR}"
  python3 -m venv "${VENV_DIR}"
fi

"${VENV_DIR}/bin/pip" install -q -r "${REQ_FILE}"

if [[ "${PLOT_ONLY}" -eq 0 ]]; then
  echo "Running benchmark suite on the current Go binaries"
  if [[ ${#SUITE_ARGS[@]} -gt 0 ]]; then
    "${VENV_DIR}/bin/python" "${SUITE_PY}" "${SUITE_ARGS[@]}"
  else
    "${VENV_DIR}/bin/python" "${SUITE_PY}"
  fi
fi

"${VENV_DIR}/bin/python" "${GEN_PY}"

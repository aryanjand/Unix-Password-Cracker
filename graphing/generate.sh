#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${SCRIPT_DIR}/.venv"
REQ_FILE="${SCRIPT_DIR}/requirements.txt"
GEN_PY="${SCRIPT_DIR}/generate.py"

if [[ ! -d "${VENV_DIR}" ]]; then
  echo "Creating virtualenv at ${VENV_DIR}"
  python3 -m venv "${VENV_DIR}"
fi

"${VENV_DIR}/bin/pip" install -q -r "${REQ_FILE}"
"${VENV_DIR}/bin/python" "${GEN_PY}"

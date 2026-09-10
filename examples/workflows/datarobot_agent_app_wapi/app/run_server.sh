AGENT_PKG_DIR="/tmp/agent-pkgs"
mkdir -p "$AGENT_PKG_DIR"

echo "Installing agent"
UV_NO_CACHE=1 uv pip install --no-deps \
    --python /app/.venv/bin/python \
    --target "$AGENT_PKG_DIR" \
    .

export PYTHONPATH="${AGENT_PKG_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

echo "Running nat serve"
exec nat dragent serve --config_file workflow.yaml --host 0.0.0.0 --port 8080 --use_gunicorn true

#!/bin/sh
set -eu

: "${OLLAMA_COMPLETION_MODEL:=qwen3:4b-instruct}"
: "${OLLAMA_EMBEDDING_MODEL:=qwen3-embedding:0.6b}"

export OLLAMA_HOST=127.0.0.1:11434

ollama serve &
server_pid=$!
trap 'kill "${server_pid}" 2>/dev/null || true; wait "${server_pid}" 2>/dev/null || true' EXIT INT TERM

attempt=0
until ollama list >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "${attempt}" -ge 60 ] || ! kill -0 "${server_pid}" 2>/dev/null; then
    echo "Ollama initialization server did not become ready." >&2
    exit 1
  fi
  sleep 1
done

for model in "${OLLAMA_COMPLETION_MODEL}" "${OLLAMA_EMBEDDING_MODEL}"; do
  if ollama show "${model}" >/dev/null 2>&1; then
    echo "Ollama model already present: ${model}"
  else
    ollama pull "${model}"
  fi
done

# The long-running server uses UID 1000 and root's group without extra capabilities.
chmod -R g+rwX /models

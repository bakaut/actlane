#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACTLANE_BIN="${ACTLANE_BIN:-$ROOT/packages/cli/dist/actlane}"
PACK="${PACK:-$ROOT/packs/create-github-draft-pr}"
TMP_ROOT="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

ensure_actlane() {
  if [[ ! -x "$ACTLANE_BIN" ]]; then
    echo "==> build actlane"
    (cd "$ROOT/packages/cli" && go build -o dist/actlane ./cmd/actlane)
  fi
  echo "==> actlane binary"
  "$ACTLANE_BIN" version
}

run_rpc() {
  local title="$1"
  local broker_bundle="$2"
  local request_file="$3"
  local output_file="$4"

  echo
  echo "## $title"
  echo "# request"
  cat "$request_file"
  echo
  echo "# response"
  "$ACTLANE_BIN" mcp serve --broker-bundle "$broker_bundle" < "$request_file" | tee "$output_file"
  if command -v jq >/dev/null 2>&1; then
    echo
    echo "# summary"
    jq -r '
      if (.result.tools? != null) then
        "tools=" + ([.result.tools[].name] | join(", "))
      elif (.result.content? != null) then
        (.result.content[0].text | fromjson) as $body |
        [
          ($body.capability // $body.delivery.capability // empty),
          ($body.policyDecision // $body.delivery.policyDecision // empty),
          ($body.evidence.id // $body.delivery.evidenceId // empty),
          ((($body.adapterExecutions // $body.delivery.adapterExecutions // []) | map(.status) | join(",")) // empty),
          ($body.execution.performed // $body.delivery.externalExecutionDone // empty),
          ($body.delivery.summary // empty)
        ] | map(tostring) | join(" | ")
      else
        empty
      end
    ' "$output_file" || true
  fi
}

write_safe_flow_requests() {
  local file="$1"
  cat > "$file" <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_classify","arguments":{"task":"Prepare a safe GitHub draft PR for reviewed README changes","changed_files":["README.md"],"branch":"main","diff_summary":"docs only update"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_load_capability","arguments":{"name":"create-github-draft-pr"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature/smoke-docs","title":"Smoke docs PR","summary":"Smoke docs update","files":["README.md"],"confirmed":true}}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"actlane_get_evidence","arguments":{"latest":true}}}
{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}
EOF
}

write_deny_flow_requests() {
  local file="$1"
  cat > "$file" <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","input":{"repo":"unknown/repo","baseBranch":"main","branch":"feature/secret","title":"Secret update","summary":"Should be denied","files":[".env"],"confirmed":false}}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_get_evidence","arguments":{"latest":true}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}
EOF
}

write_durable_flow_requests() {
  local file="$1"
  local evidence_dir="$2"
  local escaped_evidence_dir
  escaped_evidence_dir="$(json_escape "$evidence_dir")"
  cat > "$file" <<EOF
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","evidenceDir":"$escaped_evidence_dir","input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature/smoke-durable","title":"Smoke durable evidence","summary":"Smoke durable evidence","files":["README.md"],"confirmed":true}}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}
EOF
}

write_fake_mcp_server() {
  local file="$1"
  cat > "$file" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  id="$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')"
  if [[ "$line" == *'"method":"initialize"'* ]]; then
    printf '{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"fake-github-mcp","version":"smoke"}}}\n' "$id"
    continue
  fi
  if [[ "$line" == *'"method":"tools/call"'* ]]; then
    name="$(printf '%s' "$line" | sed -n 's/.*"name":"\([^"]*\)".*/\1/p')"
    printf '{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"fake-mcp:%s"}],"structuredContent":{"tool":"%s","status":"ok"}}}\n' "$id" "$name" "$name"
    continue
  fi
  printf '{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"method not found"}}\n' "${id:-null}"
done
EOF
  chmod +x "$file"
}

copy_pack_for_fake_adapter() {
  local dst="$1"
  local fake_mcp="$2"
  cp -R "$PACK" "$dst"
  rm -rf "$dst/generated"
  local binding="$dst/mcp/bindings/github-mcp-draft-pr.yaml"
  local replacement="$TMP_ROOT/fake-mcpservers.txt"
  cat > "$replacement" <<EOF
  mcpservers:
    - name: github
      provider: fake-test-mcp
      source: local-smoke-script
      transport: stdio
      command:
        - "$fake_mcp"
      env: {}
EOF
  awk -v repl="$replacement" '
    BEGIN { skipping=0 }
    /^  mcpservers:/ {
      while ((getline line < repl) > 0) print line
      close(repl)
      skipping=1
      next
    }
    /^  requiredTools:/ { skipping=0 }
    !skipping { print }
  ' "$binding" > "$binding.tmp"
  mv "$binding.tmp" "$binding"
}

write_fake_adapter_requests() {
  local file="$1"
  local evidence_dir="$2"
  local escaped_evidence_dir
  escaped_evidence_dir="$(json_escape "$evidence_dir")"
  cat > "$file" <<EOF
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","executeAdapters":true,"evidenceDir":"$escaped_evidence_dir","input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature/smoke-adapter","title":"Smoke adapter execution","summary":"Smoke adapter execution","files":["README.md"],"confirmed":true}}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_get_evidence","arguments":{"latest":true}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}
EOF
}

main() {
  ensure_actlane
  "$ACTLANE_BIN" generate "$PACK" --target codex >/dev/null

  local broker_bundle="$PACK/generated/codex/broker/broker-bundle.json"

  local safe_req="$TMP_ROOT/safe.ndjson"
  local deny_req="$TMP_ROOT/deny.ndjson"
  local durable_req="$TMP_ROOT/durable.ndjson"
  local fake_req="$TMP_ROOT/fake-adapter.ndjson"
  local durable_dir="$TMP_ROOT/durable-evidence"
  local fake_evidence_dir="$TMP_ROOT/fake-evidence"
  local fake_mcp="$TMP_ROOT/fake-github-mcp.sh"
  local fake_pack="$TMP_ROOT/create-github-draft-pr-fake"

  write_safe_flow_requests "$safe_req"
  run_rpc "safe default broker flow: classify -> load -> run -> evidence -> delivery" "$broker_bundle" "$safe_req" "$TMP_ROOT/safe.out"

  write_deny_flow_requests "$deny_req"
  run_rpc "deny flow: policy blocks secrets and final delivery requires human resolution" "$broker_bundle" "$deny_req" "$TMP_ROOT/deny.out"

  write_durable_flow_requests "$durable_req" "$durable_dir"
  run_rpc "durable evidence flow: evidenceDir writes compact JSON" "$broker_bundle" "$durable_req" "$TMP_ROOT/durable.out"
  echo
  echo "# durable evidence files"
  find "$durable_dir" -maxdepth 1 -type f -print

  write_fake_mcp_server "$fake_mcp"
  copy_pack_for_fake_adapter "$fake_pack" "$fake_mcp"
  "$ACTLANE_BIN" generate "$fake_pack" --target codex >/dev/null
  write_fake_adapter_requests "$fake_req" "$fake_evidence_dir"
  run_rpc "external adapter flow: executeAdapters=true calls fake stdio MCP" "$fake_pack/generated/codex/broker/broker-bundle.json" "$fake_req" "$TMP_ROOT/fake-adapter.out"
  echo
  echo "# fake adapter durable evidence files"
  find "$fake_evidence_dir" -maxdepth 1 -type f -print
}

main "$@"

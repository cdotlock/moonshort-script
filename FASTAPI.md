# MoonShort Script — FastAPI HTTP API

This directory provides a FastAPI HTTP wrapper around the `mss` CLI binary. An LLM (or any HTTP client) can compile, decompile, validate, and fix MSS scripts exactly as it would via the local CLI, with proper error reporting. **Every CLI subcommand has a matching HTTP endpoint.**

## Public hosted instance

> **Base URL: `http://moonshort.47.254.93.15.sslip.io:9876`**
>
> Swagger docs: http://moonshort.47.254.93.15.sslip.io:9876/docs
>
> This instance is running the `feature/fastapi-wrapper` branch in a mob sandbox on port 8888. It is exposed through a permanent `mob expose` route, not Daytona's short-lived preview URL and not bore.pub.
> The route is backed by `mob-server` in IP mode and proxies `moonshort.47.254.93.15.sslip.io:9876` to sandbox `ca4cfdc5-a605-40be-ac1a-dc0df4fbe9f8:8888`. The service guardian is configured with `/health` and restarts uvicorn if the sandbox service is down.

Smoke-tested from outside the sandbox:

```bash
BASE=http://moonshort.47.254.93.15.sslip.io:9876

curl -s "$BASE/health"
# {"status":"ok"}

curl -s -X POST "$BASE/validate" -F "script=@testdata/minimal.md"
# {"valid":true,"errors":null,"stdout":"OK"}
```

---

## Quickstart (run your own instance)

```bash
# 1. Build the binary (one time)
cd <repo-root>
go build -o bin/mss ./cmd/mss

# 2. Install Python dependencies (one time)
pip install -r requirements.txt

# 3. Start the server (keep running in background)
python -m uvicorn api_server:app --host 0.0.0.0 --port 8080 &
```

The server is now listening on `http://localhost:8080`. All endpoints return JSON.

---

## Endpoint ↔ CLI mapping

| HTTP endpoint      | CLI equivalent                                                   |
|--------------------|------------------------------------------------------------------|
| `POST /compile`    | `mss compile file.md [--assets mapping.json] -o output.json`     |
| `POST /compile-dir`| `mss compile dir/ [--assets mapping.json] -o output.json`        |
| `POST /decompile`  | `mss decompile input.json [-o output-dir]`                        |
| `POST /validate`   | `mss validate file.md [--assets mapping.json]`                    |
| `POST /fix`        | `mss fix file.md [-o output.md]`                                  |
| `GET /health`      | (server health check)                                            |

---

## Endpoints

### `GET /health`

Check whether the server is alive and the `mss` binary is found.

```bash
curl -s http://localhost:8080/health
# → {"status":"ok"}
# or
# → {"status":"unhealthy","reason":"mss binary not found"}   (HTTP 503)
```

---

### `POST /compile`

Compile a single MSS `.md` script into structured JSON.

**Request:** `multipart/form-data`

| Field    | Type     | Required | Description                          |
|----------|----------|----------|--------------------------------------|
| `script` | file     | yes      | MSS script file (`.md`), UTF-8 text  |
| `assets` | file     | no       | Asset mapping JSON (see spec below)  |

```bash
# Minimal compile (no assets)
curl -s -X POST http://localhost:8080/compile \
  -F "script=@episode.md"

# With asset mapping
curl -s -X POST http://localhost:8080/compile \
  -F "script=@episode.md" \
  -F "assets=@mapping.json"
```

**Success (200):** Returns the compiled episode JSON. Example:

```json
{
  "branch_key": "main",
  "episode_id": "main:01",
  "seq": 1,
  "title": "Test",
  "steps": [
    {"id":"0001_bg","type":"bg","name":"classroom_morning","transition":"fade","url":""},
    {"id":"0002_nar","type":"narrator","text":"Hello world."},
    {"id":"0003_you","type":"you","text":"Thinking deeply."}
  ],
  "gate": {"next": "main:02"}
}
```

**Errors:**

| HTTP status | Meaning                                          |
|-------------|--------------------------------------------------|
| 422         | Compilation failed — `detail.error` has stderr   |
| 504         | Compilation timed out (30 s)                     |
| 500         | Internal server error — `detail` has message     |

---

### `POST /compile-dir`

Compile a directory of MSS `.md` files (uploaded as a zip archive) into structured JSON. Equivalent to `mss compile <dir/>`.

**Request:** `multipart/form-data`

| Field     | Type | Required | Description                                          |
|-----------|------|----------|------------------------------------------------------|
| `zipfile` | file | yes      | Zip archive containing `.md` files (flat or nested)  |
| `assets`  | file | no       | Asset mapping JSON (same format as `/compile`)       |

```bash
# Zip up an episode directory and compile
zip -j episodes.zip 01.md 02.md 03.md
curl -s -X POST http://localhost:8080/compile-dir \
  -F "zipfile=@episodes.zip" \
  -F "assets=@mapping.json"
```

**Success (200):** Returns an array of compiled episode objects, one per `.md` file.

```json
[
  { "branch_key":"main", "episode_id":"main:01", "seq":1, "steps":[...], "gate":{...} },
  { "branch_key":"main", "episode_id":"main:02", "seq":2, "steps":[...], "gate":{...} }
]
```

**Errors:** same as `/compile` (422 / 504 / 500). Timeout is 60 s for directories.

---

### `POST /decompile`

Decompile compiled MSS JSON back into MSS source text and an asset mapping.

**Request:** `multipart/form-data`

| Field      | Type | Required | Description                          |
|------------|------|----------|--------------------------------------|
| `compiled` | file | yes      | Compiled MSS JSON file, UTF-8 text   |

```bash
curl -s -X POST http://localhost:8080/decompile \
  -F "compiled=@output.json"
```

**Success (200):**

```json
{
  "episodes": {
    "mss.md": "@episode main:01 \"Test\" {\n\n  @bg set classroom_morning fade\n  ...\n}\n"
  },
  "asset_mapping": {
    "base_url": "",
    "assets": {"bg":{},"characters":{},"music":{},"sfx":{},"cg":{},"minigames":{}}
  },
  "warnings": []
}
```

- `episodes`: map of filename → MSS source text. Typically keyed as `"mss.md"`.
- `asset_mapping`: the recovered asset mapping (may be empty/default if the original input had no assets).
- `warnings`: non-fatal diagnostics (e.g., `"unable to fully reconstruct gate X"`).

**Errors:**

| HTTP status | Meaning                                          |
|-------------|--------------------------------------------------|
| 422         | Decompilation failed — `detail.error` has stderr |
| 504         | Decompilation timed out (30 s)                   |
| 500         | Internal server error — `detail` has message     |

---

### `POST /validate`

Validate an MSS script for syntax errors without producing compiled output. Equivalent to `mss validate`.

**Request:** `multipart/form-data`

| Field    | Type | Required | Description                          |
|----------|------|----------|--------------------------------------|
| `script` | file | yes      | MSS script file (`.md`), UTF-8 text  |
| `assets` | file | no       | Asset mapping JSON (optional)        |

```bash
curl -s -X POST http://localhost:8080/validate \
  -F "script=@episode.md"
```

**Success (200) — always returns 200:**

```json
// Valid script:
{"valid":true,"errors":null,"stdout":"OK"}

// Invalid script:
{"valid":false,"errors":"error: line 1 col 10: expected IDENT, got LBRACE (\"{\")","stdout":null}
```

- `valid`: boolean — `true` means the script passes syntax validation.
- `errors`: string — the compiler's error output when `valid=false`.
- `stdout`: string — informational output (e.g., `"OK"`).

---

### `POST /fix`

Auto-fix common issues in an MSS script. Equivalent to `mss fix`. Two modes:

**Fix mode (default):** Returns the fixed script text.

```bash
curl -s -X POST http://localhost:8080/fix \
  -F "script=@broken.md"
# → {"check":false,"fixed":"@episode main:01 ...\n","changed":true,"stderr":"wrote ..."}
```

- `check`: `false`
- `fixed`: the corrected script text (string)
- `changed`: boolean — `true` if the script was modified
- `stderr`: any informational output from the fixer

**Check mode (`?check=true`):** Dry-run, only reports issues without modifying.

```bash
curl -s -X POST "http://localhost:8080/fix?check=true" \
  -F "script=@broken.md"
# → {"check":true,"issues_found":false,"report":null}
```

- `check`: `true`
- `issues_found`: boolean — `true` if the fixer found any issues
- `report`: string — diagnostic output

**Errors:**

| HTTP status | Meaning                                      |
|-------------|----------------------------------------------|
| 422         | Fix failed — `detail.error` has stderr       |
| 504         | Fix timed out (30 s)                         |
| 500         | Internal server error — `detail` has message |

---

## Asset mapping format

The `assets` parameter is a JSON file of this shape:

```json
{
  "base_url": "https://cdn.example.com/novel/",
  "assets": {
    "bg": {
      "classroom_morning": "bg/classroom_morning.png"
    },
    "characters": {
      "alice": {
        "happy": "alice_happy.png"
      }
    },
    "music": {},
    "sfx": {},
    "cg": {},
    "minigames": {}
  }
}
```

When a mapping is provided, asset `url` fields in the compiled output are resolved to full URLs (e.g., `"https://cdn.example.com/novel/bg/classroom_morning.png"`). Without a mapping, `url` fields are empty strings.

---

## End-to-end round-trip example

```bash
BASE=http://localhost:8080

# Validate first
curl -s -X POST $BASE/validate -F "script=@my_episode.md"

# Compile a script
curl -s -X POST $BASE/compile \
  -F "script=@my_episode.md" \
  -F "assets=@assets.json" \
  -o compiled.json

# Inspect the compiled JSON
cat compiled.json | python -m json.tool

# Decompile it back
curl -s -X POST $BASE/decompile \
  -F "compiled=@compiled.json" \
  -o decompiled.json

# decompiled.json contains:
#   .episodes["mss.md"]  → reconstructed MSS source
#   .asset_mapping       → recovered asset map
#   .warnings            → any non-fatal issues
```

---

## Full workflow: validate → fix → compile → decompile

```bash
BASE=http://localhost:8080

# 1. Validate
RESULT=$(curl -s -X POST $BASE/validate -F "script=@episode.md")
if ! echo "$RESULT" | python3 -c "import sys,json; sys.exit(0 if json.load(sys.stdin)['valid'] else 1)"; then
  echo "Script invalid: $(echo $RESULT | python3 -c "import sys,json; print(json.load(sys.stdin)['errors'])")"
  # 2. Try to auto-fix
  FIXED=$(curl -s -X POST $BASE/fix -F "script=@episode.md" | python3 -c "import sys,json; print(json.load(sys.stdin)['fixed'])")
  echo "$FIXED" > episode.md
  # 3. Re-validate
  curl -s -X POST $BASE/validate -F "script=@episode.md"
fi

# 4. Compile
curl -s -X POST $BASE/compile -F "script=@episode.md" -F "assets=@mapping.json" -o compiled.json

# 5. Round-trip test
curl -s -X POST $BASE/decompile -F "compiled=@compiled.json" | python3 -m json.tool
```

---

## Error handling pattern (recommended for LLM agents)

```
1. Call endpoint
2. If HTTP 2xx → success; the response body is the result
3. If HTTP 422 → read .detail.error, report the compiler error to the user, suggest fixing the script (or call /fix first)
4. If HTTP 504 → retry once; if it times out again, report the script may be too large
5. If HTTP 500 → read .detail, report the internal error
6. If connection refused → the server is not running; start it with uvicorn (see Quickstart)
```

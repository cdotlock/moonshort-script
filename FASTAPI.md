# MoonShort Script — FastAPI HTTP API

This directory provides a FastAPI HTTP wrapper around the `mss` CLI binary. An LLM (or any HTTP client) can compile MSS `.md` scripts and decompile compiled JSON exactly as it would via the local CLI, with proper error reporting.

## Quickstart (for an LLM / automated agent)

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

Compile an MSS `.md` script into structured JSON.

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

Always check the HTTP status code. On 422 the `detail.error` field contains the compiler's stderr output, which tells you exactly what went wrong (syntax error, unknown directive, etc.).

---

### `POST /decompile`

Decompile compiled JSON back into MSS source text and an asset mapping.

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

## Asset mapping format

The `assets` parameter to `/compile` is a JSON file of this shape:

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

# Decompiled result contains:
#   .episodes["mss.md"]  → reconstructed MSS source
#   .asset_mapping       → recovered asset map
#   .warnings            → any non-fatal issues
```

---

## Error handling pattern (recommended for LLM agents)

```
1. Call endpoint
2. If HTTP 2xx → print success, the response body is the result
3. If HTTP 422 → read .detail.error, report the compiler error to the user, suggest fixing the script
4. If HTTP 504 → retry once; if it times out again, report the script may be too large
5. If HTTP 500 → read .detail, report the internal error
6. If connection refused → the server is not running; start it with uvicorn (see Quickstart)
```

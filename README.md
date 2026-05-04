# moonshort-script

MoonShort Script (MSS) interpreter for MobAI interactive visual novels.

Parses `.md` script files into structured JSON for the frontend player, resolving asset semantic names to OSS URLs.

## Install

```bash
go build -o bin/mss ./cmd/mss
```

## Usage

```bash
# Compile a single episode
mss compile episode.md --assets mapping.json -o output.json

# Compile an entire novel directory
mss compile novel_001/main/ --assets mapping.json -o novel.json

# Decompile compiled JSON back to MSS + recovered asset mapping
mss decompile output.json
# writes output_decompiled/mss.md and output_decompiled/assests_mapping.json

# Validate syntax only
mss validate episode.md
```

## Script Format

See [MSS-SPEC.md](MSS-SPEC.md) for the complete specification.

## Development

```bash
make test    # Run all tests
make build   # Build binary
make package # Build dist/mss-dev-<goos>-<goarch>.tar.gz
```

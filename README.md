# moonshort-script

NoRules Script (NRS) interpreter for MobAI interactive visual novels.

Parses `.md` script files into structured JSON for the frontend player, resolving asset semantic names to OSS URLs.

## Install

```bash
go build -o bin/nrs ./cmd/nrs
```

## Usage

```bash
# Compile a single episode
nrs compile episode.md --assets mapping.yaml -o output.json

# Compile an entire novel directory
nrs compile novel_001/main/ --assets mapping.yaml -o novel.json

# Validate syntax only
nrs validate episode.md
```

## Script Format

See [NRS Script Format Design v2.1](docs/specs/2026-04-15-nrs-script-format-design.md) for the complete specification.

## Development

```bash
make test    # Run all tests
make build   # Build binary
```

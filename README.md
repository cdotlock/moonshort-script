# lunascripts

Lunascripts (LS) interpreter for MobAI interactive visual novels.

Parses `.md` script files into structured JSON for the frontend player, resolving asset semantic names to OSS URLs.

## Install

```bash
go build -o bin/lsc ./cmd/lsc
```

## Usage

```bash
# Compile a single episode
lsc compile episode.ls.md --assets mapping.json -o output.json

# Compile an entire novel directory
lsc compile novel_001/main/ --assets mapping.json -o novel.json

# Decompile compiled JSON back to LS + recovered asset mapping
lsc decompile output.json
# writes output_decompiled/episode.ls.md and output_decompiled/assets_mapping.json

# Validate syntax only
lsc validate episode.ls.md
```

## Script Format

See [LS-SPEC.md](LS-SPEC.md) for the complete specification.

## Development

```bash
make test    # Run all tests
make build   # Build binary
make package # Build dist/ls-dev-<goos>-<goarch>.tar.gz
```

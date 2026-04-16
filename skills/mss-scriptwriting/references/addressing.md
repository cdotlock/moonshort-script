# Episode ID Addressing

## Format

`<branch_key>:<seq>`

The branch_key is the directory path. The seq is the episode number within that branch.

## Main line

```
main:01          → novel/main/01.md
main:02          → novel/main/02.md
main:60          → novel/main/60.md
```

## Bad endings (1-3 episodes, terminal)

```
main/bad/001:01  → novel/main/bad/001/01.md
main/bad/001:02  → novel/main/bad/001/02.md   (last episode: no @else, or @else: @next credits)
```

## Independent routes (5-10 episodes, parallel storyline)

```
main/route/001:01  → novel/main/route/001/01.md
main/route/001:08  → novel/main/route/001/08.md
```

## Minor branches (1-3 episodes, rejoin main)

```
main/minor/11_a:01  → novel/main/minor/11_a/01.md
```

The `11_a` means "branched from episode 11, variant a".

## Remix (player-generated, session-scoped)

```
remix/abc123:01  → novel/remix/abc123/01.md
remix/abc123:02  → novel/remix/abc123/02.md
```

The session ID is a unique identifier for the remix session.

## Rules

1. Directory structure IS the branch_key — the interpreter derives the ID from the file path
2. Seq numbers are zero-padded to 2 digits: `01`, `02`, ... `60`
3. Bad endings and minor branches always originate from a `@gate` block in a main-line episode
4. Remix branches use the same format but live under `remix/` instead of `main/`
5. `@if`/`@else` targets inside `@gate` use the full `branch_key:seq` format

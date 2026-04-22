// Feature Parade BAD01 — @label/@goto + @ending bad_ending
// 路由入口: ep01 gate 的 T29a/T29b/T29c 任一命中 → 本集

@episode main/bad/001:01 "Bad Ending — Regret" {

  @bg set bedroom
  &music play night

  NARRATOR: [T36] @label declares a jump target named START.
  @label START

  NARRATOR: [T37] The door closed. You never opened it again.
  YOU: One wrong move, one closed door.

  NARRATOR: [T38] @if with flag condition drives a @goto jump below.
  @if (EP01_FACED_EASTON) {
    YOU: [T38a] Even the ones I tried for.
    NARRATOR: [T38b → label END] @goto END will fire next — the following [T39] narrator line should be SKIPPED.
    @goto END
  } @else {
    YOU: [T38c] EP01_FACED_EASTON false — no goto, fall through to [T39].
  }

  NARRATOR: [T39] SKIP CHECK: this line appears ONLY when T38's goto was NOT taken.

  @label END
  NARRATOR: [T40] @label END reached (either by fall-through or by goto jump).

  @malia look shocked
  @music fadeout
  NARRATOR: [T41] Weeks later, nothing has changed.

  NARRATOR: [T42] @ending bad_ending fires now.
  @ending bad_ending
}

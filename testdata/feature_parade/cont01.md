// Feature Parade CONT01 — @ending to_be_continued
// 路由入口: ep01 gate 的 T29d/T29e 任一命中 → 本集

@episode main/route/easton:01 "To Be Continued — Easton's Route" {

  @bg set bedroom
  &music play calm
  &easton show hopeful at right
  &malia show phone at left

  NARRATOR: [T43] Reached Easton route via compound-flag OR influence condition in EP01 gate.

  EASTON: I didn't think you'd answer.
  MALIA: I almost didn't.

  @affection easton +3
  &butterfly "Malia opened the door for Easton after EP01"

  YOU: [T44] Where does this go?

  NARRATOR: [T45] @ending to_be_continued fires — expect "next episode coming" screen.
  @ending to_be_continued
}

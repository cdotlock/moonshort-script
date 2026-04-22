// Feature Parade STRESS — 高强度逻辑压测
// 覆盖: 3 层 @if 嵌套、复合条件括号、7 种 condition 混链、空块边界、作用域交错、brave 省略 else

@episode main/stress:01 "Stress Test" {

  @bg set classroom
  &music play tense

  NARRATOR: [T50] Stress suite starts — each sub-test announces itself.
  @sfx play huh
  NARRATOR: [T50a] @sfx play huh — the "huh?" sound effect plays once.

  // ================================================================
  // [T51] 3 层 @if 嵌套
  // ================================================================
  NARRATOR: [T51] Three-level nested @if.
  @if (EP01_FACED_EASTON) {
    NARRATOR: [T51-L1] outer flag true.
    @if (affection.easton >= 3) {
      NARRATOR: [T51-L2] mid comparison true.
      @if (san >= 50) {
        NARRATOR: [T51-L3a] innermost value comparison true — 3 levels deep.
      } @else {
        NARRATOR: [T51-L3b] innermost @else (san < 50).
      }
    } @else @if (affection.easton >= 1) {
      NARRATOR: [T51-M] mid else-if (1 <= affection < 3).
    } @else {
      NARRATOR: [T51-N] mid fallback.
    }
  } @else {
    NARRATOR: [T51-O] outer @else (flag false).
  }

  // ================================================================
  // [T52] 复合条件括号优先级
  // ================================================================
  NARRATOR: [T52] Compound with parenthesized precedence: ((x||y) && (z||w)).
  @if ((A.success || B.any) && (affection.easton >= 2 || EP01_COMPLETE)) {
    NARRATOR: [T52a] outer-AND of two OR groups true.
  }

  NARRATOR: [T53] Union of two AND groups: (a && b) || (c && d).
  @if ((EP01_FACED_EASTON && san > 30) || (EP01_DEFLECTED && affection.easton < 0)) {
    NARRATOR: [T53a] either branch true.
  }

  // ================================================================
  // [T54] 同一 if 链混合全部 5 种 gate-合法 condition（check/rating 在 T55/T56 验证）
  // ================================================================
  NARRATOR: [T54] One @if chain mixing choice / flag / influence / comparison-affection / comparison-value / compound — 6 kinds.
  @if (A.fail) {
    NARRATOR: [T54a] choice condition branch.
  } @else @if (EP01_COMPLETE) {
    NARRATOR: [T54b] flag condition branch.
  } @else @if (influence "Player hesitated at every fork") {
    NARRATOR: [T54c] influence condition branch.
  } @else @if (affection.easton >= 1) {
    NARRATOR: [T54d] comparison on affection.
  } @else @if (san <= 50) {
    NARRATOR: [T54e] comparison on engine value.
  } @else @if (A.any && san > 0) {
    NARRATOR: [T54f] compound of choice && comparison.
  } @else {
    NARRATOR: [T54g] fall-through @else.
  }

  // ================================================================
  // [T55] minigame 单分支（验证 validator 允许省略 else / 其他 rating）
  // ================================================================
  NARRATOR: [T55] minigame power_swing (attr=ATK) with SINGLE rating branch, no @else — validator must accept.
  @minigame power_swing ATK "a timing-based power swing — release at peak for an S" {
    @if (rating.S) {
      NARRATOR: [T55a] Only rating.S handled — others silently pass.
    }
  }

  NARRATOR: [T59] Minigame roundup — remaining 3 games each with a single-branch pass. Attrs cover CHA/INT/WIL (ATK was in T16/T55).
  @minigame slot_machine CHA "pull the slot machine lever and read the alignment" {
    @if (rating.S) {
      NARRATOR: [T59a] slot_machine (CHA) S branch ran.
    }
  }
  @minigame stardew_fishing INT "reel in a fish with the classic bar-balance meter" {
    @if (rating.S) {
      NARRATOR: [T59b] stardew_fishing (INT) S branch ran.
    }
  }
  @minigame survive_30_seconds WIL "dodge falling hazards for 30 seconds" {
    @if (rating.S) {
      NARRATOR: [T59c] survive_30_seconds (WIL) S branch ran.
    }
  }

  // ================================================================
  // [T56] brave option 省略 @else（validator 宽松）+ safe option 嵌套 @if
  // ================================================================
  @bg set hallway fade
  &music crossfade upbeat
  &sfx play bell
  &malia show flat at center

  NARRATOR: [T56] Choice stress: brave omits @else; safe nests an @if.
  @choice {
    @option A brave "[T56-brave] Take the risk — no @else branch." {
      check {
        attr: WIL
        dc: 18
      }
      @if (check.success) {
        NARRATOR: [T56a] brave check.success — no paired @else (allowed).
        @affection easton +1
      }
    }
    @option B safe "[T56-safe] Walk away — contains nested @if." {
      @if (affection.easton >= 2) {
        YOU: [T56b] safe body nested @if true.
        @butterfly "Walked away but hesitated"
      } @else {
        YOU: [T56c] safe body nested @else.
      }
    }
  }

  // ================================================================
  // [T57] 空手机块（0 条消息）
  // ================================================================
  NARRATOR: [T57] @phone show with ZERO messages — empty block edge case.
  @phone show {
  }
  @phone hide

  // ================================================================
  // [T58] 并发组边界: 连续 @ / & / 对话交替
  // ================================================================
  @bg set gym cut
  &malia move to right
  &josie show excited at left

  NARRATOR: [T58] Concurrent-group boundary test: dialogue breaks the group above; new @ below starts a fresh group.

  @pause for 1
  NARRATOR: [T58a] @pause for 1 — single click to advance.

  @signal mark STRESS_DONE

  NARRATOR: [T60 → main:02] Stress gate has only a fallback @else — will jump to ep02.md unconditionally.
  @gate {
    @else: @next main:02
  }
}

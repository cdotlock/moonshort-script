# Feature Parade — MSS 全功能压测集

极简剧情 × 全功能覆盖 × 对白即测试清单。每个剧本里的旁白/对白都用 `[T##]` 标注当前测试点，播放时对照本文件勾选即可知进度。

## 文件结构

```
feature_parade/
├── README.md              # 本文件（含完整测试清单）
├── mapping.json           # 素材映射（指向 OSS）
├── ep01.md                # 主剧本（T1–T29）
├── ep02.md                # 补全剧本（T30–T35）+ ending: complete
├── bad01.md               # @label/@goto + ending: bad_ending（T36–T42）
├── cont01.md              # ending: to_be_continued（T43–T45）
├── stress.md              # 高强度逻辑压测（T50–T58）
├── ep01_output.json       # 编译产物（golden）
├── ep02_output.json
├── bad01_output.json
├── cont01_output.json
├── stress_output.json
└── assets/                # 素材本地副本（OSS 镜像见下）
    ├── bg/                # 6 张背景
    ├── characters/        # 6 角色 × 16 立绘
    ├── music/             # 6 首 BGM
    ├── sfx/               # 2 个音效
    ├── cg/                # 1 张 CG
    └── minigames/         # 5 个真实小游戏
        ├── parking-rush/
        ├── power-swing/
        ├── slot-machine/
        ├── stardew-fishing/
        └── survive-30-seconds/
```

## 跑通方式

```bash
make build

# 单文件
bin/mss validate testdata/feature_parade/ep01.md --assets testdata/feature_parade/mapping.json
bin/mss compile  testdata/feature_parade/ep01.md --assets testdata/feature_parade/mapping.json -o /tmp/ep01.json

# 全部重生成 golden
for f in ep01 ep02 bad01 cont01 stress; do
  bin/mss compile testdata/feature_parade/$f.md \
    --assets testdata/feature_parade/mapping.json \
    -o testdata/feature_parade/${f}_output.json
done
```

## OSS 素材

已上传至 `https://mobai-file.oss-cn-shanghai.aliyuncs.com/moonshort-script/testdata/feature_parade/`，包含：

- 6 背景 / 16 立绘 / 6 BGM / **3 SFX（bell 钟声 / crowd 狗笑 / huh "huh?"）** / **1 CG 视频（cg_example.mp4）**
- **5 个可玩小游戏**（用户提供的 minigames-example）：
  - `minigames/parking-rush/index.html` — 停车冲刺
  - `minigames/power-swing/index.html` — 时机挥击
  - `minigames/slot-machine/index.html` — 老虎机
  - `minigames/stardew-fishing/index.html` — 钓鱼
  - `minigames/survive-30-seconds/index.html` — 30 秒闪避生存

mapping.json 的 `base_url` 指向 OSS 根路径；minigame key 为下划线风格（`parking_rush`），映射到带连字符的真实文件夹（`parking-rush/`）。

---

## 测试清单（按对白编号勾选）

### EP01 —— 主剧本（T1–T29）

| # | 测试点 | 指令/特性 |
|---|---|---|
| T1 | 并发组：bg + music_play + char_show 同组 | `@` + `&` |
| T2 | 内心独白 | `YOU:` |
| T3 | 旁白 | `NARRATOR:` |
| T4 | 角色对白 | `CHARACTER:` |
| T5 | 收到消息 | `@text from` |
| T6 | 发出消息 | `@text to` |
| T7 | 手机块多条消息 | `@phone show` |
| T8 | 立绘语法糖（第 1 次） | `CHARACTER [look]:` |
| T9 | bg transition=fade + music_crossfade | `@bg set ... fade` + `@music crossfade` |
| T10 | 气泡全 9 种 | `@char bubble`：heart/anger/sweat/question/exclaim/idea/music/doom/ellipsis |
| T11 | 暂停 2 次点击 | `@pause for 2` |
| T12 | bg transition=cut + char_move + sfx | `@bg ... cut` + `@char move to` + `@sfx play` |
| T13 | 右侧对白 | `CHARACTER:` |
| T14 | 角色淡出 | `@char hide fade` |
| T15 | bg transition=slow + 多 char_look 并发 | `@bg ... slow` + `@char look` |
| T16 | minigame **parking_rush** (ATK) rating 4 档 | `@minigame` + `rating.S` / `rating.A \|\| rating.B` / `rating.C` / @else |
| T17 | 对白铺垫 | `CHARACTER:` |
| T18 | 选择块设置 | `@choice` |
| T19 | brave 选项 + check block | `@option brave` + `check { attr dc }` + `check.success` / `check.fail` |
| T20 | 状态指令组：@affection / @butterfly / @signal mark（`@signal int` 见 T58b） | `@affection ±N` / `@butterfly` / `@signal mark` |
| T21 | 成就 rarity=uncommon | `@achievement` |
| T22 | safe 选项 | `@option safe`（无 check） |
| T23 | 顶层 @if 链：compound(comparison && flag) / compound(comparison \|\| flag) / @else | `@if` + compound condition |
| T24 | CG 展示（素材是 mp4 视频） | `@cg show window_stare fade { duration content ... }` |
| T25 | influence 条件 | `@if (influence "...")` |
| T26 | engine value 比较（san） | `@if (san <= N)` / `@if (san >= N)` |
| T27 | 音乐淡出 | `@music fadeout` |
| T28 | 立绘语法糖（第 2 次） | `CHARACTER [look]:` |
| T29 | Gate 路由 5 种 condition（跳转前由 narrator 预告目的地） | 见下表 |

**T29 Gate 路由详表**（check/rating 是 context-local，不能在 gate 里，所以 gate 只能覆盖 5/7）：

真正的 `@gate` 之前放了一段**镜像 @if 链**——条件顺序与 gate 1:1 对应——在跳转发生前由 narrator 说出即将跳去哪一集，方便测试时对照：

| 分支 | 条件类型 | 条件 | 预告对白 | 去向 |
|---|---|---|---|---|
| T29a | choice | `A.fail` | `[T29a → main/bad/001:01] choice A.fail matched → will jump to bad01.md` | [bad01.md](bad01.md) |
| T29b | flag | `EP01_DEFLECTED` | `[T29b → main/bad/001:01]` | [bad01.md](bad01.md) |
| T29c | comparison | `affection.easton < 0` | `[T29c → main/bad/001:01]` | [bad01.md](bad01.md) |
| T29d | compound | `EP01_COMPLETE && affection.easton >= 3` | `[T29d → main/route/easton:01]` | [cont01.md](cont01.md) |
| T29e | influence | `influence "Player showed..."` | `[T29e → main/route/easton:01]` | [cont01.md](cont01.md) |
| T29f | fallback | `@else` | `[T29f → main/stress:01]` | [stress.md](stress.md) → [ep02.md](ep02.md) |

### 路由总览（5 个 ep 如何串起来）

```
                       ┌── [T29a/b/c] bad path ──→ bad01.md (bad_ending)
                       │
    ep01.md ──gate─────┼── [T29d/e]   route ───── → cont01.md (to_be_continued)
  (main:01)            │
                       └── [T29f]     fallback ── → stress.md ──gate──@else──→ ep02.md (complete)
                                                  (main/stress:01)    (main:02)
```

**默认路径**（所有条件都不命中时）：`ep01 → stress → ep02 (complete)` —— 按剧情跑一遍就能覆盖全部 5 个剧本。不走进 stress 的前提是 T29a–T29e 某条命中，跳去 bad01 / cont01 后剧终。

### EP02 —— 补全剧本（T30–T35）

| # | 测试点 | 指令/特性 |
|---|---|---|
| T30 | position 3 档一字排开 | `left` / `center` / `right` |
| T31 | char_look transition=dissolve | `@char look <look> dissolve` |
| T32 | 比较运算符 ==/!= | `@if (x == N)` / `@if (x != N)` |
| T33 | 比较运算符 >/< | `@if (san > N)` / `@if (san < N)` |
| T34 | 成就 rarity rare / epic / legendary | `@achievement ... { rarity: rare \| epic \| legendary }` |
| T35 | `@ending complete` | 与 @gate 互斥 |

### BAD01 —— @label / @goto + bad_ending（T36–T42）

| # | 测试点 | 指令/特性 |
|---|---|---|
| T36 | 声明跳转锚点 START | `@label START` |
| T37 | 普通剧情行 | `NARRATOR:` |
| T38 | flag 条件触发跳转（goto 前由 narrator 预告目标 label） | `@if (EP01_FACED_EASTON)` + `@goto END` |
| T39 | **跳过检查点**：仅当 T38 的 goto 没触发时才出现 | — |
| T40 | 跳转目标到达 | `@label END` |
| T41 | 终局铺垫 | `@music fadeout` + `@char look` |
| T42 | `@ending bad_ending` | 与 @gate 互斥 |

### CONT01 —— to_be_continued（T43–T45）

| # | 测试点 | 指令/特性 |
|---|---|---|
| T43 | 路由入口叙述 | `NARRATOR:` |
| T44 | 承接 | `YOU:` |
| T45 | `@ending to_be_continued` | 与 @gate 互斥 |

### STRESS —— 高强度逻辑（T50–T58）

| # | 测试点 | 指令/特性 |
|---|---|---|
| T50 | 压测起始宣告 + `@sfx play huh` (T50a) | `@sfx` 第 3 种音效 |
| T51 | 3 层 @if 嵌套（L1–L3） | `@if { @if { @if {} } }` |
| T52 | 复合条件括号优先级：`((x\|\|y) && (z\|\|w))` | `compound` |
| T53 | 复合条件并集：`(a && b) \|\| (c && d)` | `compound` |
| T54 | 一条 @if 链混合 6 种 condition | choice / flag / influence / comparison-affection / comparison-value / compound |
| T55 | minigame **power_swing** (ATK) 单分支、无 @else | `@minigame { @if (rating.S) {} }` |
| T56 | brave 省略 @else + safe 嵌套 @if | `@option brave` / `@option safe` |
| T57 | 空 `@phone show {}` 块 | 空块边界 |
| T58 | 并发组边界：对话打断 + 重启 | `&` / 对话行 |
| T58b | `@signal int` — three write forms (=, +, -) + comparison read | `@signal int stress_count = 0` / `+2` / `-1` + `@if (stress_count >= 2)` |
| T59 | 小游戏巡礼：**slot_machine (CHA) / stardew_fishing (INT) / survive_30_seconds (WIL)** | 3 个 `@minigame` 连跑 |
| T60 | stress gate 路由预告 → `main:02` | fallback `@else` only |

---

## 覆盖矩阵汇总

### AST Node（32/32 全覆盖）

Episode, GateBlock, EndingNode, LabelNode, GotoNode, PauseNode, BgSetNode, CharShowNode, CharHideNode, CharLookNode, CharMoveNode, CharBubbleNode, CgShowNode, DialogueNode, NarratorNode, YouNode, PhoneShowNode, PhoneHideNode, TextMessageNode, MusicPlayNode, MusicCrossfadeNode, MusicFadeoutNode, SfxPlayNode, MinigameNode, ChoiceNode, OptionNode, AffectionNode, SignalNode, AchievementNode, ButterflyNode, IfNode  ✓

### Condition（7/7 全覆盖）

| Condition | 覆盖位置 |
|---|---|
| `choice` | T29a gate / T54a stress |
| `flag` | T29b gate / T23 ep01 / T38 bad01 / T54b stress |
| `influence` | T25 ep01 / T29e gate / T54c stress |
| `comparison` | T23/T26 ep01 / T32/T33 ep02 / T29c gate / T54d,e stress / T58b stress (int variable) |
| `compound` | T23 ep01 / T29d gate / T52/T53/T54f stress |
| `check` | T19a/T19b ep01（context-local in brave option） |
| `rating` | T16a–d ep01 / T55a stress（context-local in minigame） |

### 属性覆盖（仅限 ATK / INT / CHA / WIL 四种）

| 属性 | 覆盖位置 |
|---|---|
| ATK | T16 parking_rush / T55 power_swing |
| CHA | T19 brave check / T59 slot_machine |
| INT | T59 stardew_fishing |
| WIL | T56 brave check / T59 survive_30_seconds |

### 枚举值（全覆盖）

| 枚举 | 覆盖 |
|---|---|
| position | T30 ep02（5 档全亮相） |
| bg transition | T9 fade / T12 cut / T15 slow / T1 default |
| char_hide transition | T14 fade |
| char_look transition | T31 dissolve |
| bubble_type | T10 ep01（9 种连发） |
| rarity | T21 uncommon / T34a rare / T34b epic / T34c legendary |
| ending type | T35 complete / T42 bad_ending / T45 to_be_continued |
| comparison op | T23 >= / T26 <= / T32 == != / T33 > < |
| compound op | T23 && / T23 \|\| / T52 / T53 |

---

## 设计原则

1. **对白即清单** —— 每个 `[T##]` 编号直接写进旁白/独白，测试者播放到哪就知道测到哪
2. **Gate 路由封闭** —— 所有 `@next` 指向本测试集内实际存在的 .md 文件
3. **剧情极简** —— 无情节铺陈，只承载功能点
4. **一次跑通** —— 5 个文件 + 一次 compile 就能回归整个 MSS 表面
5. **真实素材** —— 所有 URL 由 OSS mapping 拼接，可在前端播放器直接播出

## 维护

- 加新特性 → 在合适的剧本追加一段，给一个 `[T##]` 编号，补到本 README 的清单 + 覆盖矩阵
- Spec 变更 → `git diff testdata/feature_parade/*_output.json` 人肉审核 golden 是否符合新预期

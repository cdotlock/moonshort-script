# MSS Engine Integration Guide

> 面向前端播放器 / 后端消费端的集成指南。假设你已经读过 [JSON-OUTPUT.md](./JSON-OUTPUT.md) 了解字段形状，本文聚焦**运行时行为**——引擎怎么消费 JSON、需要维护哪些状态、怎么处理各种 step 类型和条件类型。

## 运行模型

MSS 编译器输出按集（episode）为单位的 JSON。引擎一次加载一集，按顺序遍历 `steps`，遇到选择/路由/结束时做状态决策。每集 JSON 是自包含的——跨集共享的状态（marks、affection、choice history、butterfly records、已解锁成就）由引擎维护在**持久存储**中。

### Player cursor：用 step `id` 而不是数字索引

每个 step 都带一个稳定的 `id` 字段（格式 `<seq>_<tag>`，详见 [JSON-OUTPUT.md §4.0](./JSON-OUTPUT.md)）。当后端持久化玩家进度时，**cursor 必须用完整的 ID 路径**（如 `["0005_ch", "options", "A", "steps", "0006_dlg"]`），**不要**用纯数字索引（如 `[4, "options", 0, "steps", 0]`）。原因：remix patch 可能 insert/replace 同一容器内的 step，纯数字索引会静默指错位置；ID-keyed cursor 在结构变更后仍然能 lookup 到正确节点。`id` 是冻结契约，编译器算法一旦发布即不可变更——若 backend 需要变化，必须配套写一次性 cursor migration。

引擎需要维护的跨集持久状态：

| 存储 | 写入来源 | 读取来源 |
|------|---------|---------|
| Flag store | `{type:"signal", kind:"mark", event:X}` | `{type:"flag", name:X}` 条件 |
| Affection map | `{type:"affection", character:X, delta:N}` | `{type:"comparison", left:{kind:"affection", char:X}, ...}` |
| Choice history | 选择后 | `{type:"choice", option:X, result:Y}` 条件 |
| Butterfly records | `{type:"butterfly", description:X}` | `{type:"influence", description:X}` 条件（经 LLM 求值） |
| Engine values (san / CHA / etc.) | 引擎内部规则 | `{type:"comparison", left:{kind:"value", name:X}, ...}` |
| Unlocked achievements | `{type:"achievement", achievement_id, name, rarity, description}` | — 纯通知，不反向查询 |

集内临时状态（单集生命周期内有效）：

- **当前 brave option 的 check 结果** — 玩家点选 brave option → 引擎掷 D20 + 属性修正，判定 success/fail → 进入 option 的 `steps`，求值 `check.success/fail` 条件

`@minigame` 不参与 D20 检定（无属性绑定）。`@trick` 不写任何状态（无评级、无奖励）。两者都不在临时状态里出现。

## Step 消费规则

顶层 `steps` 是混合数组，元素可能是对象（单步）或数组（并发组）。遍历逻辑：

```
for element in steps:
    if element is array:          # 并发组
        execute_all_concurrently(element)
        if group_contains_click_wait(element):
            wait_for_player_click()
    else:                          # 单步
        dispatch(element)          # 按 element.type 分支
        if is_click_wait(element.type):
            wait_for_player_click()
```

**点击等待的 step 类型**：`dialogue` / `narrator` / `you` / `pause`（`pause.clicks` 指定等待几次）。

**阻塞交互的 step 类型**：`choice`（等待玩家选择）、`minigame`（等待游戏结果或玩家跳过）、`trick`（必须等玩家完成；不可跳）、`phone_show` / `phone_hide`（UI 切换）、`cg_show`（放视频）、`if`（引擎判定后执行对应分支）。

**自动推进的 step 类型**：其余全部——`bg` / `char_show` / `char_hide` / `char_look` / `char_move` / `bubble` / `music_play` / `music_crossfade` / `music_fadeout` / `sfx_play` / `affection` / `signal` / `achievement` / `butterfly` / `label` / `goto`。

## 关键 step 类型

### `signal`

```json
{"type": "signal", "kind": "mark", "event": "EVENT_NAME"}
```

当前 kind 只有 `"mark"`——写入持久 flag store。`kind` 字段保留在 JSON 里，方便未来扩展新 kind 时不破坏消费者：**引擎见到未知 kind 应当忽略并记日志**，不要崩溃、不要写入。

后续条件求值 `{"type":"flag","name":"EVENT_NAME"}` 时在 flag store 中查找即可。

### `achievement`

```json
{
  "type": "achievement",
  "id": "HIGH_HEEL_DOUBLE_KILL",
  "name": "Heel Twice Over",
  "rarity": "epic",
  "description": "Once is improvisation. Twice is a signature move."
}
```

Step 自带完整元数据——引擎不需要另外的声明表查找，消费时：

1. 检查 `id` 是否已在持久 unlock store 中：
   - 是 → 跳过（不重复弹窗）
   - 否 → 写入 unlock store、弹 UI、推数据上报
2. 不写 flag store——`@if (HIGH_HEEL_DOUBLE_KILL)` 永远 false（成就是外发通知）

剧本侧的形态：`@achievement <id> { name / rarity / description }`。条件触发由外层 `@if` 承担，典型写法 `@if (mark1 && mark2) { @achievement X { ... } }`——只有所有前置 mark 已在 flag store 时才会执行到 achievement step。

### `choice` → brave option

```json
{
  "id": "A",
  "mode": "brave",
  "text": "Stand your ground.",
  "check": {"attr": "CHA", "dc": 12},
  "steps": [
    {
      "type": "if",
      "condition": {"type": "check", "result": "success"},
      "then": [...],
      "else": [...]
    }
  ]
}
```

消费流程：

1. 玩家点选 option A（brave）
2. 引擎依据 `check.attr` 和 `check.dc` 掷 D20（仅加属性修正）
3. 判定 success/fail，**存入当前集的临时 check context**
4. 进入 option A 的 `steps` 数组
5. 遍历到 `if` step 时，求值 `condition`：
   - `{type:"check", result:"success"}` → 当临时 check context == success 时返回 true
   - `{type:"check", result:"fail"}` → 当临时 check context == fail 时返回 true
6. 走 `then` 或 `else`，没有对应分支（如作者省略 `else`）就直接跳过该 if

**通用性**：option.steps 可能完全不含 check 条件——safe option 就是这样，body 直接是叙事。brave option 的 body 里也可能有 check 之外的其他 step（普通 if、narrator、char_show 等），混杂着 check 条件 if。引擎统一按普通 step 遍历即可，`check` 只是一种 condition type。

**check context 生命周期**：从玩家确定选择到遍历完该 option 的 steps 为止。遍历结束后 check context 应清理（防止后续步骤里不相关的 `check.*` 条件意外求值成 true）。

### `minigame` (可选)

```json
{
  "type": "minigame",
  "name": "qte_challenge",
  "game_url": "https://.../qte_challenge/index.html",
  "description": "Mr. Chen pairs Malia and Mauricio for the opening reading exercise — they trade lines from Jane Eyre..."
}
```

消费流程：

1. 显示 UI（"play" 与 "skip" 两个入口）。
2. 玩家选 **skip** → 当本条 step 完成，立即推进到下一个 step；不写任何状态、不发奖励。
3. 玩家选 **play** → 载入小游戏（`game_url`），阻塞主剧情，等 H5 回传分数。
4. 引擎按分数缩放奖励并写入持久状态（金币、SAN、XP 等——具体规则由引擎决定，**反作弊在此**）。
5. 关闭小游戏 UI，推进到下一个 step。

**字段使用说明**：

- **`name`**：素材句柄，与 `assets.minigames.<name>` 对应。
- **`game_url`**：解析后的 H5 入口；生成器尚未跑完时可能为空字符串或缺省——播放器应当能优雅降级（显示 "尚未生成" 占位）。
- **`description`**：连贯英文 prose——既描述场景，也定义简单玩法。给下游 **vibe-coding agent** 生成 H5 用；运行时播放器可忽略。

### `trick` (强制)

```json
{
  "type": "trick",
  "trick_type": "hold",
  "prompt": "Hold your breath until he walks past."
}
```

消费流程：

1. 引擎读取 `trick_type`，启动对应的原生检测器（触摸 / 运动 / 摄像头几何与状态）。
2. 在主剧情上方叠加 `prompt` 文案的浮层。
3. **阻塞**直到玩家完成动作。玩家不能 skip。
4. 完成 → 移除浮层，推进到下一个 step。

**`trick_type` 锁定值**（共 6 个）：

| trick_type | 模态 | 权限 | 引擎应实现的检测 |
|---|---|---|---|
| `tap` | 触摸 | 无 | 在 N 秒内累计 N 次点击 |
| `hold` | 触摸 | 无 | 按住屏幕至少 N 秒不放 |
| `swipe` | 触摸 | 无 | 任意方向滑动一次（不跟路径） |
| `shake` | 运动 | 无运行时弹窗 | 陀螺仪 / 加速度计阈值 |
| `swing` | 运动 | 无运行时弹窗 | 单次大幅摆动手势 |
| `hold-still` | 运动 | 无运行时弹窗 | 陀螺仪几乎不动持续 N 秒 |

**阈值由引擎拥有**——脚本不传，未来也不允许传。所有 6 个类型都不需要摄像头 / 麦克风权限。

**未知 `trick_type` 的处理**：编译器侧 parser/validator 会先拒绝；万一旧版引擎遇到生疏的 type（例如未来扩展），应当记日志并把该 step 当作 1 秒延时跳过，不要崩溃。但**永远不要**允许玩家手动 skip——trick 是强制原语，这是它与 minigame 的本质差别。

### `cg_show`

```json
{
  "type": "cg_show",
  "name": "window_stare",
  "url": "https://.../window_stare.png",
  "transition": "fade",
  "duration": "medium",
  "content": "The camera opens on Malia's silhouette against the rain-streaked window...",
  "steps": [
    {"type": "you", "text": "The city lights blurred through my tears."}
  ]
}
```

**谁消费哪些字段**：

- **agent-forge（视频管线）**：`duration` 档位（low/medium/high）和 `content` 叙事是视频生成的输入。管线据此调度镜头、生成视频文件，把输出 URL 写进素材映射。
- **运行时播放器**：只播放 `url` 指向的视频文件（或图片，如果还没过 agent-forge）。`duration` 和 `content` 可忽略——但可以作为预检查字段（url 未生成时根据 content 显示文本 fallback）。
- **body steps**：CG 放映期间叠加在视频上的对白/叙事。播放器按正常 step 消费。

## 条件类型

`if` step 和 gate route 的 `condition` 字段都是结构化对象。完整类型见 JSON-OUTPUT.md §4.8；这里只强调 context-local 的两个：

| type | 字段 | 作用域 | 求值 |
|------|------|-------|------|
| `check` | `result: "success" \| "fail"` | 当前 brave option 的 steps 内 | 读临时 check context |

`check` 是 context-local——引擎在遍历 brave option 的 steps 时维护临时 check context，遍历结束清理。

**不做作用域静态校验**——编译器允许作者把 `{"type":"check"}` 条件写在任何位置。如果不在 brave option 体内求值，引擎就没 check context，**返回 false 即可**（不要崩溃）。这是设计意图：作用域错放是作者的剧情 bug，不是语法错。

## 常见消费逻辑示例

### 处理 signal step

```pseudocode
function handle_signal(step):
    kind = step.kind
    event = step.event
    switch kind:
        case "mark":
            flag_store.insert(event)
        default:
            log_warning("unknown signal kind", kind)  # forward-compat
```

### 处理 achievement step

```pseudocode
function handle_achievement(step):
    # NOTE: step.achievement_id is the semantic id from MSS source
    # (`@achievement <id>`). step.id is the cursor stable step id
    # (`<seq>_ach`) used by the player cursor — do not confuse the two.
    if unlock_store.contains(step.achievement_id):
        return  # 已解锁，不重复
    unlock_store.insert(step.achievement_id)
    ui.show_achievement_popup(step.name, step.rarity, step.description)
    analytics.report_achievement_unlock(step.achievement_id, step.rarity)
```

### 处理 if step（条件求值）

```pseudocode
function evaluate_condition(cond, ctx):
    switch cond.type:
        case "flag":
            return flag_store.contains(cond.name)
        case "comparison":
            left = read_operand(cond.left, ctx)
            return compare(left, cond.op, cond.right)
        case "compound":
            l = evaluate_condition(cond.left, ctx)
            r = evaluate_condition(cond.right, ctx)
            return cond.op == "&&" ? (l && r) : (l || r)
        case "choice":
            hist = choice_history.lookup(cond.option)
            if hist is null: return false
            if cond.result == "any": return true
            return hist.result == cond.result
        case "influence":
            return llm_judge(butterfly_records, cond.description)
        case "check":
            if ctx.check is null: return false  # 作用域外
            return ctx.check.result == cond.result
```

`ctx` 是引擎维护的当前集临时状态，包含 `ctx.check`（当前 brave option 的检定结果，离开 option 体时清空）。

### 处理 choice → brave option

```pseudocode
function handle_choice(step):
    pick = ui.show_options(step.options)
    opt = step.options[pick]
    if opt.mode == "brave":
        # D20 公式：roll + 属性修正。
        roll = d20() + attribute_bonus(opt.check.attr)
        result = (roll >= opt.check.dc) ? "success" : "fail"
        ctx.check = {result: result}
        choice_history.insert(opt.id, result)
        walk_steps(opt.steps, ctx)
        ctx.check = null
    else:  # safe
        choice_history.insert(opt.id, "any")   # safe 记作 "any"（用于 A.any 条件查询）
        walk_steps(opt.steps, ctx)
```

### 处理 minigame（可选）

```pseudocode
function handle_minigame(step):
    action = ui.show_minigame_offer(step.name, step.description)  # play / skip
    if action == "skip":
        return  # 整条 step no-op，下一个 step 直接执行
    score = ui.play_minigame(step.game_url)         # 阻塞，等 H5 回传分数
    rewards = scale_rewards(score)                  # 反作弊在此
    apply_rewards(rewards)                          # 引擎写持久状态
```

### 处理 trick（强制）

```pseudocode
function handle_trick(step):
    detector = engine.start_trick_detector(step.trick_type)  # 6 选 1
    ui.show_prompt_overlay(step.prompt)
    detector.await_completion()                              # 阻塞；玩家不能跳
    ui.hide_prompt_overlay()
    # 无奖励、无 ctx 写入、无评级 — trick 的唯一作用是制造"在场感"
```

# MSS Engine Integration Guide (v2.4)

> 面向前端播放器 / 后端消费端的集成指南。假设你已经读过 [JSON-OUTPUT.md](./JSON-OUTPUT.md) 了解字段形状，本文聚焦**运行时行为**——引擎怎么消费 JSON、需要维护哪些状态、怎么处理各种 step 类型和条件类型。

## 运行模型

MSS 编译器输出按集（episode）为单位的 JSON。引擎一次加载一集，按顺序遍历 `steps`，遇到选择/路由/结束时做状态决策。每集 JSON 是自包含的——跨集共享的状态（marks、affection、choice history、butterfly records、已解锁成就）由引擎维护在**持久存储**中。

引擎需要维护的跨集持久状态：

| 存储 | 写入来源 | 读取来源 |
|------|---------|---------|
| Flag store | `{type:"signal", kind:"mark", event:X}` | `{type:"flag", name:X}` 条件 |
| Affection map | `{type:"affection", character:X, delta:N}` | `{type:"comparison", left:{kind:"affection", char:X}, ...}` |
| Choice history | 选择后 | `{type:"choice", option:X, result:Y}` 条件 |
| Butterfly records | `{type:"butterfly", description:X}` | `{type:"influence", description:X}` 条件（经 LLM 求值） |
| Engine values (san / CHA / etc.) | 引擎内部规则 | `{type:"comparison", left:{kind:"value", name:X}, ...}` |
| Unlocked achievements | `{type:"achievement", id:X}` | — 纯通知，不反向查询 |

集内临时状态（单集生命周期内有效）：

- **当前 brave option 的 check 结果** — 玩家点选 brave option → 引擎掷 D20 + 属性修正 + 小游戏修正，判定 success/fail → 进入 option 的 `steps`，求值 `check.success/fail` 条件
- **当前 minigame 的 rating** — 玩家玩完小游戏 → 引擎得到评级 → 进入 minigame 的 `steps`，求值 `rating.<grade>` 条件

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

**阻塞交互的 step 类型**：`choice`（等待玩家选择）、`minigame`（等待游戏结果）、`phone_show` / `phone_hide`（UI 切换）、`cg_show`（放视频）、`if`（引擎判定后执行对应分支）。

**自动推进的 step 类型**：其余全部——`bg` / `char_show` / `char_hide` / `char_look` / `char_move` / `bubble` / `music_play` / `music_crossfade` / `music_fadeout` / `sfx_play` / `affection` / `signal` / `achievement` / `butterfly` / `label` / `goto`。

## v2.4 关键变更

### 1. Signal 只有一种 kind

`{type:"signal", kind:"mark", event:"EVENT_NAME"}` 写入持久 flag store。后续条件求值：
```json
{"type": "flag", "name": "EVENT_NAME"}
```
→ 在 store 中查找 → 找到返回 true。

**向前兼容**：`kind` 字段会一直存在，哪怕目前只有 `"mark"` 一种值。引擎见到未知 kind 应当**忽略并记日志**——不要崩溃，不要写入。将来语言会引入新 kind（如 `once` 一次性标记、`counter` 计数器等），老引擎未升级前应该能安全跳过。

**成就触发不再是 signal**——独立的 `achievement` step 类型，见下条。

### 2. 独立的 `achievement` step 类型

```json
{"type": "achievement", "id": "HIGH_HEEL_DOUBLE_KILL"}
```

消费流程：

1. 在当前 episode 的 `achievements` 数组里查 id == X 的条目（未找到 = 声明缺失，log warning 并跳过）
2. 检查该成就是否已在持久存储中标记为 unlocked：
   - 是 → 跳过（不重复弹窗）
   - 否 → 走成就系统：写入 unlocked store、弹 UI、推数据上报
3. 不写 flag store——`@if (HIGH_HEEL_DOUBLE_KILL)` 永远是 false（这是设计意图，成就是外发通知）

**剧本侧的触发形态**：剧本用 `@achievement <id>`（无 `{`）生成这类 step。通常包裹在 `@if (mark1 && mark2) { @achievement X }` 里——只有所有前置 mark 都已写入 flag store 时才会真正执行 achievement step。

**声明来自 `achievements` 数组**：顶层 `achievements: [{id, name, rarity, description}]`（v2.4 无 `when` 字段）。引擎加载一集时应将这些声明合并到全局注册表；跨集引用成立的前提是引擎已加载声明。

### 3. Choice/brave 的 check 分支

**v2.4 改动**：brave option 不再有 `on_success` / `on_fail` 字段；**统一在 `steps` 里**。

新形态：
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
2. 引擎依据 `check.attr` 和 `check.dc` 掷 D20（加上属性修正和最近一次 minigame 修正，如有）
3. 判定 success/fail，**存入当前集的临时 check context**
4. 进入 option A 的 `steps` 数组
5. 遍历到 `if` step 时，求值 `condition`：
   - `{type:"check", result:"success"}` → 当临时 check context == success 时返回 true
   - `{type:"check", result:"fail"}` → 当临时 check context == fail 时返回 true
6. 走 `then` 或 `else`，没有对应分支（如作者省略 `else`）就直接跳过该 if

**完全通用性**：option.steps 也可能完全不含 check 条件——safe option 就是这样，body 直接是叙事。brave option 的 body 里也可能有 check 之外的其他 step（普通 if、narrator、char_show 等），混杂着 check 条件 if。引擎统一按普通 step 遍历即可，`check` 只是一种新的 condition type。

**check context 生命周期**：从玩家确定选择到遍历完该 option 的 steps 为止。遍历结束后 check context 应清理（防止后续步骤里不相关的 `check.*` 条件意外求值成 true）。

### 4. Minigame 的 rating 分支

**v2.4 改动**：minigame 不再有 `on_results` 字段；**统一在 `steps` 里**，多加一个 `description` 字段和一套 `rating` 条件类型。

新形态：
```json
{
  "type": "minigame",
  "game_id": "qte_challenge",
  "game_url": "https://.../qte_challenge/index.html",
  "attr": "ATK",
  "description": "a quick hallway partner-assignment check where Malia matches Mauricio's beat or drops the rhythm",
  "steps": [
    {
      "type": "if",
      "condition": {"type": "rating", "grade": "S"},
      "then": [...],
      "else": {
        "type": "if",
        "condition": {
          "type": "compound",
          "op": "||",
          "left":  {"type": "rating", "grade": "A"},
          "right": {"type": "rating", "grade": "B"}
        },
        "then": [...],
        "else": [...]
      }
    }
  ]
}
```

消费流程：

1. 载入小游戏（从 `game_url`），阻塞主剧情
2. 玩家玩完 → 引擎得到评级（典型是 S/A/B/C/D，但 language 不强制字典）
3. 评级**存入当前集的临时 rating context**，并**作为 D20 修正写入临时 check modifier**（用于下一次 brave option 的 check）
4. 进入 minigame 的 `steps`
5. 遍历到 `if` step 时，求值：
   - `{type:"rating", grade:"S"}` → 当临时 rating == "S" 时返回 true
6. 走对应分支

**description 字段**：给下游视觉/视频管线用（agent-forge 生成场景过渡、解说等）。纯运行时播放器**可以忽略** description——但如果你做的是素材预处理 pipeline，这是必填输入。

**minigame 体可以完全为空**（`steps: []` 或字段缺失）——作者没有 per-rating 额外叙事时的合法形态。

**rating context 生命周期**：从小游戏结束到遍历完 minigame.steps 为止。check modifier（评级转化的数值）单独进入临时 check context，用于下一次 brave option 的掷骰——这个 modifier 的生命周期通常是"到下一次 check 生效为止"。

### 5. CG 的 duration / content

**v2.4 新增**：`cg_show` 步骤多了两个字段：

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

**谁消费**：

- **agent-forge（视频管线）**：`duration` 档位（low/medium/high）和 `content` 叙事是视频生成的输入。管线据此调度镜头、生成视频文件，把输出 URL 写进素材映射。
- **运行时播放器**：只播放 `url` 指向的视频文件（或图片，如果还没过 agent-forge）。`duration` 和 `content` 可忽略——但可以作为预检查字段（url 未生成时根据 content 显示文本 fallback）。
- **body steps**：CG 放映期间叠加在视频上的对白/叙事。播放器按正常 step 消费。

### 6. 新增条件类型：check / rating

`emitCondition` 输出里会出现两种新 type：

| type | 字段 | 含义 |
|------|------|------|
| `check` | `result: "success" \| "fail"` | 当前 brave option 的 D20 检定结果 |
| `rating` | `grade: string` | 当前 minigame 的评级（任意标识符，典型是 S/A/B/C/D 单字母） |

这两种是 **context-local**——引擎在遍历 brave option 的 steps 时维护临时 check context，遍历 minigame.steps 时维护临时 rating context。遍历结束后两个 context 清理。

**不做作用域静态校验**——编译器允许作者把 `{"type":"check"}` 条件写在任何位置。如果不在 brave option 体内求值，引擎就没 check context，**返回 false 即可**（不要崩溃）。这是设计意图：作用域错放是作者的剧情 bug，不是语法错。

同理 `rating` 在非 minigame 体内恒为 false。

### 7. Achievements 数组：不再有 `when`

**v2.4 改动**：

```json
"achievements": [
  {
    "id": "HIGH_HEEL_DOUBLE_KILL",
    "name": "Heel Twice Over",
    "rarity": "epic",
    "description": "Once is improvisation. Twice is a signature move."
  }
]
```

**旧引擎如果还在读 `when` 字段**：不会崩（字段不存在时 JSON 解析返回默认值），但会错失自动触发。**必须配合迁移**：旧引擎的 "watch marks, auto-evaluate when" 逻辑可以删除——改为：成就的触发现在纯粹通过 `{type:"achievement", id:X}` step 推过来，引擎被动响应即可。

这个变化大幅简化了引擎：

- **不再需要**成就 watcher（监听 flag store 变化、重新求值所有成就的 when 条件）
- **不再需要**集加载时的成就自动求值（看是否加载时就有旧 marks 能触发 when）
- **新做法**：引擎只负责存成就声明、响应 achievement step（查 id → 弹 UI → 记 unlock）

触发时机完全由剧本控制——剧作者把 `@if (...) { @achievement X }` 放在他想要触发的剧情点。

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
    id = step.id
    decl = achievement_registry.lookup(id)
    if decl is null:
        log_warning("achievement triggered but not declared", id)
        return
    if unlock_store.contains(id):
        return  # 已解锁，不重复
    unlock_store.insert(id)
    ui.show_achievement_popup(decl.name, decl.rarity, decl.description)
    analytics.report_achievement_unlock(id, decl.rarity)
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
        case "rating":
            if ctx.rating is null: return false  # 作用域外
            return ctx.rating.grade == cond.grade
```

`ctx` 是引擎维护的当前集临时状态，包含 `ctx.check`（当前 brave option 的检定结果，离开 option 体时清空）和 `ctx.rating`（当前 minigame 的评级，离开 minigame.steps 时清空）。

### 处理 choice → brave option

```pseudocode
function handle_choice(step):
    pick = ui.show_options(step.options)
    opt = step.options[pick]
    if opt.mode == "brave":
        modifier = consume_check_modifier()   # 之前 minigame 攒下的修正，一次性
        roll = d20() + attribute_bonus(opt.check.attr) + modifier
        result = (roll >= opt.check.dc) ? "success" : "fail"
        ctx.check = {result: result}
        choice_history.insert(opt.id, result)
        walk_steps(opt.steps, ctx)
        ctx.check = null
    else:  # safe
        choice_history.insert(opt.id, "any")   # safe 记作 "any"（用于 A.any 条件查询）
        walk_steps(opt.steps, ctx)
```

### 处理 minigame

```pseudocode
function handle_minigame(step):
    rating = ui.play_minigame(step.game_url)   # 阻塞
    ctx.rating = {grade: rating}
    set_check_modifier(rating_to_modifier(rating))   # 攒给下一次 brave 检定
    walk_steps(step.steps, ctx)
    ctx.rating = null
```

## 迁移 checklist（从 v2.3 引擎 → v2.4 引擎）

- [ ] **Signal**：移除对 `kind == "achievement"` 的处理代码；该路径改走独立的 `achievement` step
- [ ] **Signal**：增加 `default` 分支对未来未知 kind 兜底（log + skip）
- [ ] **Achievement step**：新增对 `{type:"achievement", id:X}` 的处理——查声明、写 unlock、弹 UI、推数据
- [ ] **Achievement declarations**：`achievements` 数组里的条目不再有 `when` 字段；移除 "watch marks, auto-evaluate when" 逻辑
- [ ] **Choice brave**：不再读 `option.on_success` / `option.on_fail`；改为遍历 `option.steps`，期间维护 `ctx.check` 让 `check` 条件可求值
- [ ] **Minigame**：不再读 `minigame.on_results` map；改为遍历 `minigame.steps`，期间维护 `ctx.rating` 让 `rating` 条件可求值
- [ ] **CG**：如果引擎消费了 CG 字段（纯播放器一般不动），接受新的 `duration`（low/medium/high）和 `content`（string）字段。agent-forge 管线侧须强依赖这两字段
- [ ] **Conditions**：在条件求值 switch 里新增 `case "check"` 和 `case "rating"`——作用域外返回 false，不崩
- [ ] **Minigame description**：播放器侧可忽略；素材/视觉管线需据此为 minigame 提供场景文本

## 版本协商

`episode_id` 和 JSON 结构本身没有版本字段——这是有意的：每次集编译都是"当前版本"的完整快照，引擎不需要版本分支。一次性迁移完后就不再需要同时支持 v2.3 和 v2.4 了。如果分阶段上线，建议**编译器和引擎同步升级**——别让旧引擎读新 JSON（会丢成就触发、错过 check/rating 分支），也别让新引擎读老 JSON（会找不到 `steps` 字段，找得到 `on_success` 却不消费）。

如果必须同时支持两个版本的 JSON，运行时做 shape 侦测：存在 `option.on_success` = v2.3，存在 `option.steps` = v2.4。但这是权宜之计，不建议长期维护。

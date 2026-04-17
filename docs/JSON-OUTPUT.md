# MSS JSON 输出参考手册

> 供前端播放器/游戏引擎对接使用。描述 `mss compile` 输出的 JSON 结构、步骤类型、并发分组规则和引擎消费逻辑。

---

## 1. 顶层结构

```json
{
  "episode_id": "main:01",
  "branch_key": "main",
  "seq": 1,
  "title": "Butterfly",
  "steps": [ ... ],
  "gate": { ... },
  "ending": null,
  "achievements": []
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `episode_id` | string | 集的完整标识，格式 `<branch_key>:<seq>`，如 `"main:01"` |
| `branch_key` | string | 分支路径，如 `"main"`、`"main/bad/001"`、`"remix/abc123"` |
| `seq` | number | 集序号（从 1 开始） |
| `title` | string | 集标题 |
| `steps` | array | 步骤数组（混合类型，见下文） |
| `gate` | object\|null | 路由规则（嵌套 if/else 链）。**始终存在**：路由集为 object，终结集为 `null` |
| `ending` | object\|null | 终结标记。**始终存在**：终结集为 `{"type": "complete"\|"to_be_continued"\|"bad_ending"}`，路由集为 `null` |
| `achievements` | array | 该集声明的成就表（空时为 `[]`）。每项：`{id, name, rarity, description, when}`，详见 §8 |

**`gate` 与 `ending` 互斥**：编译器保证两者不会同时为非 null。路由集用 `gate`，终结集用 `ending`，任意集必有其一。

---

## 2. Steps 数组：混合类型

`steps` 数组包含两种元素类型：

```
steps: [
  { ... },          // 对象 → 单步骤
  [ {...}, {...} ],  // 数组 → 并发组
  { ... },          // 对象 → 单步骤
  ...
]
```

### 2.1 单步骤（对象）

一个普通的 JSON 对象，代表一条独立指令。引擎按顺序执行。

```json
{
  "type": "narrator",
  "text": "Senior year. Day one."
}
```

### 2.2 并发组（数组）

一个 JSON 数组，包含多个步骤对象。引擎**同时执行**组内所有步骤。

```json
[
  {
    "type": "bg",
    "name": "school_hallway",
    "url": "https://oss.mobai.com/.../school_hallway.png",
    "transition": "fade"
  },
  {
    "type": "music_crossfade",
    "name": "tense_strings",
    "url": "https://oss.mobai.com/.../tense_strings.mp3"
  },
  {
    "type": "char_show",
    "character": "mauricio",
    "look": "neutral_smirk",
    "url": "https://oss.mobai.com/.../mauricio_neutral_smirk.png",
    "position": "right"
  }
]
```

### 2.3 分组规则

并发组由 MSS 脚本中的 `@`（领导者）和 `&`（跟随者）前缀决定：

- `@` 指令开启新的步骤组
- `&` 指令加入前一个步骤组
- 对话行（dialogue / narrator / you）始终独立
- 只有一条指令的组自动展平为对象，不包裹数组

| MSS 脚本 | JSON 输出 |
|----------|----------|
| `@bg set ...` 单独 | `{ "type": "bg", ... }` 对象 |
| `@bg set ...` + `&music ...` + `&char ...` | `[{ "type": "bg" }, { "type": "music_play" }, { "type": "char_show" }]` 数组 |
| `NARRATOR: text` | `{ "type": "narrator", ... }` 对象（始终独立） |

---

## 3. 引擎消费规则

### 3.1 遍历算法

```
for element in steps:
    if element is array:
        // 并发组：同时执行所有步骤
        execute_all_simultaneously(element)
        if any_click_wait_type(element):
            wait_for_player_click()
    else:
        // 单步骤
        execute(element)
        if is_click_wait_type(element):
            wait_for_player_click()
```

### 3.2 点击等待类型

以下 type 需要等待玩家点击后才推进到下一个步骤：

| type | 说明 |
|------|------|
| `dialogue` | 角色对白 |
| `narrator` | 旁白 |
| `you` | 内心独白 |
| `pause` | 显式暂停（等待 clicks 次点击） |

### 3.3 自动推进类型

以下 type 执行后立即推进，不等待玩家输入：

| type | 说明 |
|------|------|
| `bg` | 背景切换 |
| `char_show` | 角色入场 |
| `char_hide` | 角色退场 |
| `char_look` | 换立绘 |
| `char_move` | 角色移位 |
| `bubble` | 气泡动画 |
| `music_play` | 播放 BGM |
| `music_crossfade` | 交叉淡入 BGM |
| `music_fadeout` | 淡出 BGM |
| `sfx_play` | 播放音效 |
| `affection` | 好感度变更 |
| `signal` | 事件信号（kind=mark 持久布尔标记 / kind=achievement 触发成就） |
| `butterfly` | 蝴蝶效应记录 |
| `label` | 跳转锚点（引擎内部用） |
| `goto` | 跳转（引擎内部用） |

### 3.4 交互阻塞类型

以下 type 会阻塞流程，由引擎内部管理推进：

| type | 说明 |
|------|------|
| `choice` | 选择菜单，等待玩家选择 |
| `minigame` | 小游戏，等待游戏结果 |
| `phone_show` | 手机界面 |
| `phone_hide` | 关闭手机 |
| `cg_show` | CG 展示（内含子步骤） |
| `if` | 条件分支（引擎判定后执行对应分支） |

---

## 4. 步骤类型完整参考

### 4.1 视觉类

#### `bg` — 背景切换

```json
{
  "type": "bg",
  "name": "school_hallway",
  "url": "https://oss.mobai.com/.../school_hallway.png",
  "transition": "fade"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 素材语义名 |
| `url` | string | 是 | 已解析的 OSS URL |
| `transition` | string | 否 | `"fade"` / `"cut"` / `"slow"`，不写为交叉溶解 |

#### `char_show` — 角色入场

```json
{
  "type": "char_show",
  "character": "mauricio",
  "look": "neutral_smirk",
  "url": "https://oss.mobai.com/.../mauricio_neutral_smirk.png",
  "position": "right",
  "transition": "fade"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID（小写） |
| `look` | string | 是 | 立绘名 |
| `url` | string | 是 | 已解析的 OSS URL |
| `position` | string | 是 | `"left"` / `"center"` / `"right"` / `"left_far"` / `"right_far"` |
| `transition` | string | 否 | 入场过渡效果 |

#### `char_hide` — 角色退场

```json
{
  "type": "char_hide",
  "character": "mauricio",
  "transition": "fade"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID |
| `transition` | string | 否 | 退场过渡效果 |

#### `char_look` — 换立绘

```json
{
  "type": "char_look",
  "character": "mauricio",
  "look": "arms_crossed_angry",
  "url": "https://oss.mobai.com/.../mauricio_arms_crossed_angry.png",
  "transition": "dissolve"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID |
| `look` | string | 是 | 新立绘名 |
| `url` | string | 是 | 已解析的 OSS URL |
| `transition` | string | 否 | `"dissolve"` 等过渡效果 |

#### `char_move` — 角色移位

```json
{
  "type": "char_move",
  "character": "mauricio",
  "position": "left"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID |
| `position` | string | 是 | 目标位置 |

#### `bubble` — 气泡动画

```json
{
  "type": "bubble",
  "character": "josie",
  "bubble_type": "heart"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID |
| `bubble_type` | string | 是 | `"anger"` / `"sweat"` / `"heart"` / `"question"` / `"exclaim"` / `"idea"` / `"music"` / `"doom"` / `"ellipsis"` |

#### `cg_show` — CG 展示

```json
{
  "type": "cg_show",
  "name": "window_stare",
  "url": "https://oss.mobai.com/.../window_stare.png",
  "transition": "fade",
  "steps": [
    {"type": "you", "text": "The city lights blurred through my tears."}
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | CG 素材语义名 |
| `url` | string | 是 | 已解析的 OSS URL |
| `transition` | string | 否 | 过渡效果 |
| `steps` | array | 否 | CG 展示期间执行的子步骤 |

### 4.2 对话类

#### `dialogue` — 角色对白

```json
{
  "type": "dialogue",
  "character": "mauricio",
  "text": "Hey, Butterfly."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `character` | string | 是 | 角色 ID（**始终小写**，脚本中 `MAURICIO:` → JSON 中 `"mauricio"`） |
| `text` | string | 是 | 对白内容 |

> **注意：** 所有步骤类型中的 `character` 字段在 JSON 输出中统一为小写，包括 `dialogue`、`text_message`、`char_show`、`char_look`、`char_hide`、`char_move`、`bubble`、`affection` 等。

#### `narrator` — 旁白

```json
{
  "type": "narrator",
  "text": "Senior year. Day one."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `text` | string | 是 | 旁白内容 |

#### `you` — 内心独白

```json
{
  "type": "you",
  "text": "Another year. Same mess."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `text` | string | 是 | 独白内容 |

### 4.3 时序控制类

#### `pause` — 暂停等待

```json
{
  "type": "pause",
  "clicks": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `clicks` | number | 是 | 等待玩家点击的次数 |

### 4.4 手机/消息类

#### `phone_show` — 打开手机界面

```json
{
  "type": "phone_show",
  "messages": [
    {"type": "text_message", "direction": "from", "character": "easton", "text": "Can we talk?"},
    {"type": "text_message", "direction": "to", "character": "mauricio", "text": "How do you know?"}
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `messages` | array | 否 | 消息列表 |

#### `phone_hide` — 关闭手机界面

```json
{
  "type": "phone_hide"
}
```

无额外字段。

#### `text_message` — 短信消息（phone_show 内部）

```json
{
  "type": "text_message",
  "direction": "from",
  "character": "easton",
  "text": "Can we talk? I miss you."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `direction` | string | 是 | `"from"`（收到）/ `"to"`（发出） |
| `character` | string | 是 | 角色 ID |
| `text` | string | 是 | 消息内容 |

### 4.5 音频类

#### `music_play` — 播放 BGM

```json
{
  "type": "music_play",
  "name": "calm_morning",
  "url": "https://oss.mobai.com/.../calm_morning.mp3"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 曲目语义名 |
| `url` | string | 是 | 已解析的 OSS URL |

#### `music_crossfade` — 交叉淡入新 BGM

```json
{
  "type": "music_crossfade",
  "name": "tense_strings",
  "url": "https://oss.mobai.com/.../tense_strings.mp3"
}
```

字段同 `music_play`。

#### `music_fadeout` — 淡出停止

```json
{
  "type": "music_fadeout"
}
```

无额外字段。

#### `sfx_play` — 播放音效

```json
{
  "type": "sfx_play",
  "name": "door_slam",
  "url": "https://oss.mobai.com/.../door_slam.mp3"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 音效语义名 |
| `url` | string | 是 | 已解析的 OSS URL |

### 4.6 游戏机制类

#### `choice` — 选择菜单

```json
{
  "type": "choice",
  "options": [
    {
      "id": "A",
      "mode": "brave",
      "text": "Stand your ground.",
      "check": { "attr": "CHA", "dc": 12 },
      "on_success": [ ... ],
      "on_fail": [ ... ]
    },
    {
      "id": "B",
      "mode": "safe",
      "text": "Have Mark make a scene.",
      "steps": [ ... ]
    }
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `options` | array | 是 | 选项列表 |

**Option 对象：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 选项编号（A / B / C ...） |
| `mode` | string | 是 | `"brave"`（需检定）/ `"safe"`（无检定） |
| `text` | string | 是 | 选项显示文本 |
| `check` | object | brave 必填 | `{ "attr": "CHA", "dc": 12 }` |
| `on_success` | array | brave 必填 | 检定成功后执行的步骤 |
| `on_fail` | array | brave 必填 | 检定失败后执行的步骤 |
| `steps` | array | safe 必填 | 选择后执行的步骤 |

#### `minigame` — 小游戏

```json
{
  "type": "minigame",
  "game_id": "qte_challenge",
  "game_url": "https://oss.mobai.com/.../qte_challenge/index.html",
  "attr": "ATK",
  "on_results": {
    "S": [ ... ],
    "A B": [ ... ],
    "C D": [ ... ]
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `game_id` | string | 是 | 小游戏 ID |
| `game_url` | string | 是 | 已解析的小游戏 URL |
| `attr` | string | 是 | 关联属性名 |
| `on_results` | object | 是 | 评级 → 步骤映射，key 为空格分隔的评级（如 `"A B"`） |

### 4.7 状态变更类

#### `affection` — 好感度

```json
{ "type": "affection", "character": "easton", "delta": 2 }
```

#### `signal` — 事件信号

```json
{ "type": "signal", "kind": "mark", "event": "EP01_COMPLETE" }
{ "type": "signal", "kind": "achievement", "event": "HIGH_HEEL_WARRIOR" }
```

`kind` 字段区分 signal 的两种用途：

| kind | 后端语义 |
|------|---------|
| `mark` | 持久布尔标记。引擎永久存储，作为玩家状态。后续脚本中的 `@if (NAME)` 条件（JSON 里表现为 `{"type":"flag","name":"NAME"}`）直接查询此存储。用于触发隐藏剧情线、条件对白、arc 成就判定。 |
| `achievement` | 成就触发事件。直接引用已声明的 `@achievement` ID。引擎走成就系统：弹窗、解锁、推数据。**不**进入 flag 存储，`@if` 条件查不到。 |

#### `mark` 生命周期

1. 脚本发出 `@signal mark EVENT_NAME`
2. 引擎收到 `{"type": "signal", "kind": "mark", "event": "EVENT_NAME"}`，写入持久存储
3. 后续脚本中 `@if (EVENT_NAME) { }` 编译为 `{"type": "if", "condition": {"type": "flag", "name": "EVENT_NAME"}, ...}`
4. 引擎求值：在已存储的 marks 中查找 → 找到返回 true，否则返回 false

示例：

```
// Episode 1: 打标记
@signal mark MINIGAME_PERFECT_EP01
```
→ JSON: `{"type": "signal", "kind": "mark", "event": "MINIGAME_PERFECT_EP01"}`

```
// Episode 3: 查询
@if (MINIGAME_PERFECT_EP01) {
  NARRATOR: You remember that perfect run.
}
```
→ JSON: `{"type": "if", "condition": {"type": "flag", "name": "MINIGAME_PERFECT_EP01"}, "then": [...]}`

#### `achievement` 触发

```
@signal achievement FIRST_KISS
```
→ JSON: `{"type": "signal", "kind": "achievement", "event": "FIRST_KISS"}`

引擎收到该 signal 后，查找 episode JSON 顶层 `achievements` 数组中 `id == "FIRST_KISS"` 的条目，触发对应的成就（弹窗、UI、分析上报）。`achievement` signal **不**影响任何 flag 存储，`@if (FIRST_KISS)` 查不到。

#### `butterfly` — 蝴蝶效应

```json
{ "type": "butterfly", "description": "在食堂正面接受了Easton的靠近" }
```

| 类型 | 字段 | 说明 |
|------|------|------|
| `affection` | `character: string, delta: number` | 角色好感变化量 |
| `signal` | `kind: string, event: string` | `kind` 为 `mark`（持久布尔标记）或 `achievement`（成就触发） |
| `butterfly` | `description: string` | 蝴蝶效应描述 |

### 4.8 流程控制类

#### `if` — 条件分支

条件为**完全结构化的 AST 对象**（不含表达式字符串），后端可直接遍历判定。`else` 可以是步骤数组（简单 else）或嵌套的 `if` 对象（`@else @if` 链）。

```json
{
  "type": "if",
  "condition": {
    "type": "compound",
    "op": "&&",
    "left":  {"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 5},
    "right": {"type": "comparison", "left": {"kind": "value", "name": "CHA"}, "op": ">=", "right": 14}
  },
  "then": [ ... ],
  "else": {
    "type": "if",
    "condition": {"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 3},
    "then": [ ... ],
    "else": [ ... ]
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `condition` | object | 是 | 结构化条件 AST（见条件类型参考） |
| `then` | array | 是 | 条件成立时执行的步骤 |
| `else` | array\|object | 否 | 步骤数组（简单 `@else { }` → `[...]`）或裸 `if` 对象（`@else @if` 链 → `{"type": "if", ...}`）。前端需按类型判别：数组表示终端 else 分支，对象表示继续链式判定 |

#### 条件类型参考

条件对象包含 `type` 字段和类型特有字段。**所有条件字段都是结构化的，不含表达式字符串**——后端不需要再次解析。

**`choice`** — 选项检定结果

```json
{"type": "choice", "option": "A", "result": "fail"}
```

| 字段 | 取值 |
|------|------|
| `option` | 选项 ID（`"A"` / `"B"` / ...） |
| `result` | `"success"` / `"fail"` / `"any"` |

**`flag`** — 信号布尔标记

```json
{"type": "flag", "name": "EP01_COMPLETE"}
```

**`influence`** — LLM 蝴蝶效应判定

```json
{"type": "influence", "description": "玩家展现过对Easton的持续接纳"}
```

**`comparison`** — 数值比较。左侧结构化为 `affection.<char>` 或 `<value_name>`，右侧为整数：

```json
{"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 5}
{"type": "comparison", "left": {"kind": "value", "name": "san"}, "op": "<=", "right": 20}
```

| 字段 | 说明 |
|------|------|
| `left.kind` | `"affection"` 或 `"value"` |
| `left.char` | `kind=="affection"` 时携带，角色 id（小写） |
| `left.name` | `kind=="value"` 时携带，引擎管理的数值名（如 `"san"`、`"CHA"`） |
| `op` | `">="` / `"<="` / `">"` / `"<"` / `"=="` / `"!="` |
| `right` | 整数 |

**`compound`** — 复合条件。`left` 和 `right` 是递归的完整条件对象：

```json
{
  "type": "compound",
  "op": "&&",
  "left":  {"type": "flag", "name": "A"},
  "right": {"type": "comparison", "left": {"kind": "value", "name": "san"}, "op": "<=", "right": 20}
}
```

| 字段 | 说明 |
|------|------|
| `op` | `"&&"` 或 `"\|\|"` |
| `left` | 左子条件（任意条件类型对象，支持递归嵌套） |
| `right` | 右子条件（任意条件类型对象，支持递归嵌套） |

#### `label` — 跳转锚点

```json
{ "type": "label", "name": "AFTER_FIGHT" }
```

#### `goto` — 跳转

```json
{ "type": "goto", "target": "AFTER_FIGHT" }
```

---

## 5. 并发组行为

### 5.1 执行模型

```
并发组 = [step_1, step_2, step_3]

引擎行为：
  1. 同时发起 step_1, step_2, step_3 的执行
  2. 检查组内是否包含点击等待类型（dialogue / narrator / you / pause）
     - 有 → 渲染完成后等待玩家点击
     - 无 → 立即推进到下一个步骤/组
```

### 5.2 典型并发组场景

**场景搭建**（最常见用法）：

MSS 源码：
```
@bg set school_cafeteria fade
&music crossfade casual_lunch
&mark show grin_confident at right
&malia show neutral_flat at left
```

JSON 输出：
```json
[
  {"type": "bg", "name": "school_cafeteria", "url": "...", "transition": "fade"},
  {"type": "music_crossfade", "name": "casual_lunch", "url": "..."},
  {"type": "char_show", "character": "mark", "look": "grin_confident", "url": "...", "position": "right"},
  {"type": "char_show", "character": "malia", "look": "neutral_flat", "url": "...", "position": "left"}
]
```

引擎行为：同时切背景 + 换音乐 + 两个角色入场。全部是自动推进类型，无需等待。

**多角色同时动作**：

```json
[
  {"type": "char_look", "character": "mauricio", "look": "arms_crossed_angry", "url": "..."},
  {"type": "bubble", "character": "josie", "bubble_type": "sweat"}
]
```

引擎行为：Mauricio 换表情和 Josie 冒汗同时发生。

---

## 6. Gate 路由

`gate` 是集末尾的路由声明，决定玩家完成本集后跳转到哪一集。采用嵌套 if/else 链结构，条件为结构化对象。

```json
"gate": {
  "if": {"type": "choice", "option": "A", "result": "fail"},
  "next": "main/bad/001:01",
  "else": {
    "if": {"type": "influence", "description": "玩家展现过对Easton的持续接纳"},
    "next": "main/route/001:01",
    "else": {"next": "main:02"}
  }
}
```

### 6.1 判定规则

- 引擎按 if → else.if → else 链式判定
- 第一个命中的条件生效，使用对应的 `next` 跳转
- 最内层的 `else` 只有 `next` 字段，为兜底路线（fallback）

### 6.2 条件类型

Gate 中的条件使用与 body `@if` 相同的结构化 AST 格式（见 4.8 条件类型参考），所有五种类型均可使用：

| type | 说明 | JSON 示例 |
|------|------|----------|
| `choice` | 选项检定结果 | `{"type": "choice", "option": "A", "result": "fail"}` |
| `flag` | 信号布尔标记 | `{"type": "flag", "name": "EP01_COMPLETE"}` |
| `comparison` | 数值比较 | `{"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 5}` |
| `influence` | LLM 蝴蝶效应判定 | `{"type": "influence", "description": "玩家展现过..."}` |
| `compound` | 复合条件（递归） | `{"type": "compound", "op": "\|\|", "left": {...}, "right": {...}}` |

### 6.3 Gate 节点结构

每个 gate 节点包含以下字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `if` | object | 否 | 结构化条件对象（兜底节点无此字段） |
| `next` | string | 是 | 条件命中时跳转的目标 episode_id |
| `else` | object | 否 | 下一个 gate 节点（嵌套 if/else）或仅含 `next` 的兜底节点 |

---

## 7. Ending 终结标记

`ending` 字段是集的终结声明，与 `gate` 互斥——一集要么继续路由，要么终结。

```json
"ending": {"type": "bad_ending"}
```

| type | 含义 | 引擎行为建议 |
|------|------|-------------|
| `complete` | 全剧终 | 滚字幕/致谢画面，可回到主菜单 |
| `to_be_continued` | 待续 | "本章完，敬请期待" 画面 |
| `bad_ending` | 坏结局 | 显示坏结局提示，提供重开本章入口 |

**消费规则：**

- 若 `ending != null`：播放完 `steps` 后进入终结画面，不再消费 `gate`
- 若 `gate != null`：播放完 `steps` 后按 `gate` 判定跳转
- 两者均为 null 在编译时已拒绝，前端不会遇到

---

## 8. 完整 JSON 示例

以下示例展示了并发组、单步骤、暂停、选择、条件分支和路由的完整 JSON 输出：

```json
{
  "episode_id": "main:01",
  "branch_key": "main",
  "seq": 1,
  "title": "Butterfly",
  "steps": [
    [
      {
        "type": "bg",
        "name": "malias_bedroom_morning",
        "url": "https://oss.mobai.com/novel_001/bg/malias_bedroom_morning.png"
      },
      {
        "type": "music_play",
        "name": "calm_morning",
        "url": "https://oss.mobai.com/novel_001/music/calm_morning.mp3"
      },
      {
        "type": "char_show",
        "character": "malia",
        "look": "neutral_phone",
        "url": "https://oss.mobai.com/novel_001/characters/malia_neutral_phone.png",
        "position": "left"
      }
    ],
    {
      "type": "narrator",
      "text": "Senior year. Day one. Status: already complicated."
    },
    {
      "type": "you",
      "text": "Another year. Same mess."
    },
    {
      "type": "phone_show",
      "messages": [
        {"type": "text_message", "direction": "from", "character": "easton", "text": "Can we talk? I miss you."},
        {"type": "text_message", "direction": "from", "character": "easton", "text": "I know I messed up."}
      ]
    },
    {
      "type": "phone_hide"
    },
    {
      "type": "you",
      "text": "Eight months and he still won't stop."
    },
    {
      "type": "char_look",
      "character": "malia",
      "look": "worried",
      "url": "https://oss.mobai.com/novel_001/characters/malia_worried.png"
    },
    {
      "type": "pause",
      "clicks": 1
    },
    [
      {
        "type": "bg",
        "name": "school_front",
        "url": "https://oss.mobai.com/novel_001/bg/school_front.png",
        "transition": "fade"
      },
      {
        "type": "music_crossfade",
        "name": "upbeat_school",
        "url": "https://oss.mobai.com/novel_001/music/upbeat_school.mp3"
      },
      {
        "type": "char_show",
        "character": "malia",
        "look": "neutral_flat",
        "url": "https://oss.mobai.com/novel_001/characters/malia_neutral_flat.png",
        "position": "left"
      },
      {
        "type": "char_show",
        "character": "josie",
        "look": "cheerful_wave",
        "url": "https://oss.mobai.com/novel_001/characters/josie_cheerful_wave.png",
        "position": "right"
      }
    ],
    {
      "type": "dialogue",
      "character": "josie",
      "text": "Malia! Over here!"
    },
    {
      "type": "bubble",
      "character": "josie",
      "bubble_type": "heart"
    },
    {
      "type": "dialogue",
      "character": "malia",
      "text": "Hey, Jo."
    },
    {
      "type": "choice",
      "options": [
        {
          "id": "A",
          "mode": "brave",
          "text": "Let him come.",
          "check": {
            "attr": "CHA",
            "dc": 12
          },
          "on_success": [
            {
              "type": "char_look",
              "character": "easton",
              "look": "relieved",
              "url": "https://oss.mobai.com/novel_001/characters/easton_relieved.png"
            },
            {"type": "dialogue", "character": "easton", "text": "Can I sit?"},
            {"type": "dialogue", "character": "malia", "text": "You have two minutes."},
            {"type": "affection", "character": "easton", "delta": 2},
            {"type": "butterfly", "description": "在食堂正面接受了Easton的靠近"}
          ],
          "on_fail": [
            {
              "type": "char_look",
              "character": "easton",
              "look": "hurt",
              "url": "https://oss.mobai.com/novel_001/characters/easton_hurt.png"
            },
            {"type": "dialogue", "character": "malia", "text": "I... I can't do this."},
            {"type": "butterfly", "description": "试图面对Easton但失去了勇气"}
          ]
        },
        {
          "id": "B",
          "mode": "safe",
          "text": "Have Mark make a scene.",
          "steps": [
            {
              "type": "char_show",
              "character": "mark",
              "look": "grin_mischief",
              "url": "https://oss.mobai.com/novel_001/characters/mark_grin_mischief.png",
              "position": "right"
            },
            {"type": "char_hide", "character": "easton", "transition": "fade"},
            {"type": "dialogue", "character": "mark", "text": "HEY EASTON! You want some of my mystery casserole?"},
            {"type": "bubble", "character": "mark", "bubble_type": "music"},
            {"type": "you", "text": "Thank god for Mark."},
            {"type": "butterfly", "description": "让Mark帮忙避开了和Easton的正面接触"}
          ]
        }
      ]
    },
    {
      "type": "if",
      "condition": {
        "type": "compound",
        "op": "&&",
        "left":  {"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 5},
        "right": {"type": "comparison", "left": {"kind": "value", "name": "CHA"}, "op": ">=", "right": 14}
      },
      "then": [
        {"type": "dialogue", "character": "easton", "text": "You remembered."}
      ],
      "else": [
        {"type": "dialogue", "character": "easton", "text": "...Hey."}
      ]
    },
    {"type": "signal", "kind": "mark", "event": "EP01_COMPLETE"}
  ],
  "gate": {
    "if": {"type": "choice", "option": "A", "result": "fail"},
    "next": "main/bad/001:01",
    "else": {
      "if": {"type": "influence", "description": "玩家展现过对Easton的持续接纳"},
      "next": "main/route/001:01",
      "else": {"next": "main:02"}
    }
  },
  "ending": null,
  "achievements": [
    {
      "id": "HEEL_WARRIOR",
      "name": "【高跟鞋战士】",
      "rarity": "rare",
      "description": "用高跟鞋当武器，一次是即兴，签名招式就此诞生。",
      "when": {"type": "flag", "name": "HIGH_HEEL_EP01"}
    }
  ]
}
```

### 7.1 终结集示例

```json
{
  "episode_id": "main/bad/001:02",
  "branch_key": "main/bad/001",
  "seq": 2,
  "title": "Bad End",
  "steps": [
    {"type": "bg", "name": "malias_bedroom_night", "url": "..."},
    {"type": "narrator", "text": "She never came home."}
  ],
  "gate": null,
  "ending": {"type": "bad_ending"},
  "achievements": []
}
```

---

## 8. Achievements 成就表

顶层 `achievements` 字段为当前 episode 声明的成就定义列表，**恒为数组**（无成就时为 `[]`）。字段结构对齐 [`cdotlock/story-achievement-generator`](https://github.com/cdotlock/story-achievement-generator) 的 skill 输出 schema，后端可直接消费：

```json
"achievements": [
  {
    "id": "HIGH_HEEL_DOUBLE_KILL",
    "name": "【高跟鞋双杀】",
    "rarity": "epic",
    "description": "用高跟鞋当武器，一次是即兴，两次是签名招式。",
    "when": {
      "type": "compound",
      "op": "&&",
      "left":  {"type": "flag", "name": "HIGH_HEEL_EP05"},
      "right": {"type": "flag", "name": "HIGH_HEEL_EP24"}
    }
  }
]
```

### 8.1 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 成就唯一标识符。可被 `@signal achievement <id>` 引用 |
| `name` | string | 是 | 显示名称 |
| `rarity` | string | 是 | `"uncommon"` / `"rare"` / `"epic"` / `"legendary"`（**无 `common`**） |
| `description` | string | 是 | DM 口吻 flavor 文本 |
| `when` | object | 是 | 触发条件，结构化 Condition AST（与 `@if` / gate 条件完全同结构） |

### 8.2 触发方式

后端可采用两种触发机制，二选一或并用：

1. **声明式（推荐）**：监听 `mark` signal 存储的变化，每当有 mark 被写入或更新，重新求值所有未解锁成就的 `when` 条件——满足则解锁。适合跨集 arc 成就。
2. **命令式**：脚本用 `@signal achievement <id>` 直接触发某个成就。后端直接解锁 `achievements[*].id == event` 的成就，跳过 `when` 判定。适合需要精确时序的单集成就。

两种方式都必须先通过顶层 `achievements` 数组加载声明——没有声明的 id 被 signal 引用时视为错误。

### 8.3 稀有度目标分布

| rarity | 典型解锁率 | 典型触发形态 |
|--------|-----------|-------------|
| `uncommon` | 20-40% | 单集戏剧性选择的其中一支 |
| `rare` | 5-20% | 反直觉选择、DC 14+ 检定、两集组合、失败路径 |
| `epic` | 1-5% | 3+ 集的行为模式 |
| `legendary` | <1% | 4+ 集的组合 + 特定检定结果 |

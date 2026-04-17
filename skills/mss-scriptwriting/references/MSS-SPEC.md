# MoonShort Script (MSS) 格式设计规范 v2.3

> 定义 MobAI 互动视觉小说的完整脚本标记语法、文件结构、解释器职责。

---

## 1. 设计目标

1. **单一格式统一叙事与游戏机制**：一个 `.md` 文件既是剧情脚本也是数值定义
2. **自包含**：每个文件包含一集所需的全部信息，不依赖外部清单
3. **LLM 友好**：Dramatizer 和 Remix Executor 都由 LLM 生成此格式，语法对 LLM 自然、无认知负担
4. **高效解析**：Go 单二进制解释器消费，输出结构化 JSON 供前端播放器使用
5. **素材解耦**：脚本只写语义名，解释器通过独立的素材映射表翻译为 OSS URL

---

## 2. 文件结构

### 2.1 文件粒度

一个 `.md` 文件 = 一集。Dramatizer 按集输出，Remix Executor 按集或按选择后片段输出。

### 2.2 文件后缀

使用 `.md` 而非自定义后缀。虽然内容不是标准 Markdown，但 `.md` 随处可打开、GitHub 可预览、无需自定义编辑器支持。

### 2.3 文件命名

对齐 Dramatizer 的 episode_id 寻址体系，路径即 ID：

```
novel_<id>/
├── main/
│   ├── 01.md                      # main:01
│   ├── 02.md                      # main:02
│   ├── ...
│   ├── bad/
│   │   ├── 001/
│   │   │   ├── 01.md              # main/bad/001:01
│   │   │   └── 02.md              # main/bad/001:02
│   │   └── 002/
│   │       └── 01.md              # main/bad/002:01
│   ├── route/
│   │   └── 001/
│   │       ├── 01.md              # main/route/001:01
│   │       └── 02.md              # main/route/001:02
│   └── minor/
│       └── 11_a/
│           └── 01.md              # main/minor/11_a:01
└── remix/
    └── <session_id>/
        ├── 01.md                  # remix/<session_id>:01
        └── 02.md                  # remix/<session_id>:02
```

**规则**：
- 目录结构即 `branch_key`，文件名（不含扩展名）即 `seq`
- 解释器从路径推导 `episode_id`（如 `main/bad/001/02.md` → `main/bad/001:02`）
- Remix 输出放在 `remix/<session_id>/` 下，格式与主线完全一致

### 2.4 文件编码

UTF-8，换行符 `\n`。

---

## 3. 语法总览

### 3.1 基本规则

```
// 这是注释，行首 // 标记
// 空行被忽略

@episode main:01 "Butterfly" {
  // 所有内容在 @episode 块内
  // 指令以 @ 开头
  // 对话以 角色名: 开头（全大写）
  // 块用 { } 界定，支持嵌套
}
```

- **注释**：`//` 行首标记，整行忽略
- **空行**：忽略，用于可读性分段
- **字符串**：用双引号 `"..."` 包裹
- **块**：`{ }` 界定，可嵌套
- **指令**：`@` 前缀，格式为 `@<对象> <行动> [参数]`
- **并发指令**：`&` 前缀，与前一条 `@` 指令同时执行（详见 4.9 并发控制）
- **对话**：`角色名: 文本`，角色名全大写，无 `@` 前缀

### 3.2 指令分类

| 类别 | 职责 |
|------|------|
| 结构控制 | 集定义、路由、跳转 |
| 视觉呈现 | 背景、角色、CG、气泡 |
| 对话 | 角色对白、旁白、内心独白 |
| 手机/消息 | 短信界面 |
| 音频 | BGM、音效 |
| 游戏机制 | 小游戏、选择、D20 检定 |
| 状态变更 | 好感度、信号、蝴蝶效应 |
| 流程控制 | 条件分支、标签跳转 |
| 时序控制 | 暂停等待、并发执行 |

---

## 4. 指令完整定义

### 4.1 结构控制

#### `@episode <branch_key> <title> { }`

集定义，整个文件的根块。

```
@episode main:01 "Butterfly" {
  // 全部内容
}
```

#### `@gate { }`

路由声明块，**必须位于 `@episode` 块的尾部**。引擎按 `@if` → `@else @if` → `@else` 链式判定，第一个命中的条件生效；全不命中则走 `@else`。

```
@gate {
  @if (A.fail): @next main/bad/001:01
  @else @if ("玩家展现过对Easton的持续接纳"): @next main/route/001:01
  @else: @next main:02
}
```

`@gate` 与 `@ending` 互斥——一集要么继续路由，要么终结。

#### `@ending <type>`

终结标记，**与 `@gate` 二选一**。用于声明一集是剧情的终点，引擎据此展示结束画面或"敬请期待"提示。三种类型：

| type | 含义 | 典型用法 |
|------|------|---------|
| `complete` | 全剧终 | 主线大结局、所有角色 Happy End |
| `to_be_continued` | 待续 | 本季/本章已完，下一章尚未写好 |
| `bad_ending` | 坏结局 | 坏路线终点，NPC 阵亡、玩家出局等 |

```
@episode main/bad/001:02 "Bad End" {
  NARRATOR: She never came home.
  @ending bad_ending
}
```

**约束**：
- 每集最多一个 `@ending`
- `@ending` 和 `@gate` 不可同时出现（前者表示终结，后者表示继续路由）
- 若既无 `@gate` 也无 `@ending`，validator 报 `MISSING_TERMINAL` 错误

#### `@if (<condition>): @next <branch_key>`

单条跳转规则。括号 `()` 是必需的。五种条件类型，全部被编译器解析为**结构化 AST**（后端直接消费，无需再次解析表达式字符串）：

**类型一：choice（直接根据选择跳转）**
```
@if (A.fail): @next main/bad/001:01
```

源语法：`<option_id>.<check_result>`（点号分隔）
- option_id：选项编号（A / B / C 等，对应 `@option` 的 ID）
- check_result：`success` | `fail` | `any`（`any` 表示无论检定成功或失败均匹配）

**类型二：influence（LLM 检定蝴蝶效应）**
```
@if (influence "玩家多次展现对弱者的同情心"): @next main/route/001:01
@if ("玩家多次展现对弱者的同情心"): @next main/route/001:01
```

两种写法均合法：`@if (influence "description")` 和 `@if ("description")`（裸字符串）。运行时引擎将玩家积累的所有 `@butterfly` 记录喂给 LLM，判断是否满足条件。

**类型三：flag（信号布尔标记）**
```
@if (EP01_COMPLETE): @next main/route/002:01
```

单个标识符，引擎查询该 signal 是否被触发过。

**类型四：comparison（数值比较）**

左侧支持两种结构：
- `affection.<char>`：角色好感度
- `<name>`：引擎管理的数值（如 `san`、`CHA`）

右侧必须是整数字面量。合法操作符：`>=`、`<=`、`>`、`<`、`==`、`!=`。

```
@if (affection.easton >= 5): @next main/route/001:01
@if (san <= 20): @next main/bad/002:01
```

**类型五：compound（复合条件）**

通过 `&&`（与）、`||`（或）组合任意其他条件类型。支持括号分组。优先级：`||` 低于 `&&`。

```
@if (san <= 20 || FAILED_TWICE): @next main/bad/002:01
@if ((A.success || B.success) && affection.easton >= 3): @next main/route/002:01
```

#### `@else @if (<condition>): @next <branch_key>`

链式条件，前一个 `@if` 不命中时继续判定。可连续使用多个 `@else @if`。

#### `@else: @next <branch_key>`

兜底路线，所有 `@if` / `@else @if` 都不命中时走这里。

#### `@label <name>` / `@goto <name>`

集内跳转锚点。**高级指令，慎用**。

```
@if (SKIP_FIGHT) {
  @goto AFTER_FIGHT
}

// ... 打斗内容 ...

@label AFTER_FIGHT
NARRATOR: The dust settled.
```

#### `@pause for <N>`

暂停指令，等待玩家点击 N 次后继续推进。用于需要玩家主动确认才继续的节奏控制点。

```
@pause for 1       // 等待玩家点击一次
@pause for 3       // 等待玩家点击三次（用于长停顿/情绪渲染）
```

---

### 4.2 视觉呈现

所有视觉指令遵循 **对象-行动** 模式：`@<对象> <行动> [参数]`。

#### 角色指令

**`@<char> show <look> at <pos>`** — 角色入场

```
@mauricio show neutral_smirk at right
```

- `char`：角色 ID（小写）
- `look`：立绘名，对应素材语义名 `{char}_{look}`
- `pos`：`left` | `center` | `right` | `left_far` | `right_far`

| 位置 | 屏幕位置 | 适合场景 |
|------|---------|---------|
| `left` | 左侧 1/4 处 | 对话一方 |
| `center` | 正中间 | 独白、特写 |
| `right` | 右侧 1/4 处 | 对话另一方 |
| `left_far` | 最左边缘 | 第三角色 / 远处 |
| `right_far` | 最右边缘 | 第三角色 / 远处 |

**`@<char> hide [transition]`** — 角色退场

```
@mauricio hide
@mauricio hide fade
```

**`@<char> look <look> [transition]`** — 换立绘

```
@mauricio look arms_crossed_angry
@mauricio look arms_crossed_angry dissolve
```

- 不写 transition = 瞬切
- `dissolve` = 0.3s 交叉溶解

**`@<char> move to <pos>`** — 角色移位

```
@mauricio move to left
```

**`@<char> bubble <type>`** — 气泡动画

一次性播放后自动消失，跟随角色，角色退场时自动清除。

| type | 视觉效果 |
|------|---------|
| `anger` | 💢 怒气 |
| `sweat` | 💧 冷汗 |
| `heart` | ❤️ 心动 |
| `question` | ❓ 疑惑 |
| `exclaim` | ❗ 惊讶 |
| `idea` | 💡 灵机一动 |
| `music` | 🎵 开心/哼歌 |
| `doom` | 💀 绝望 |
| `ellipsis` | ... 沉默/无语 |

#### 背景指令

**`@bg set <name> [transition]`** — 切换背景

| transition | 效果 | 耗时 | 适用场景 |
|-----------|------|------|---------|
| （不写） | 交叉溶解 | 0.5s | 同场所内时间变化 |
| `fade` | 先黑屏再淡入 | 1.0s | 场景切换 |
| `cut` | 直切 | 0s | 快速跳转 |
| `slow` | 慢速淡入 | 2.0s | 情绪性转场 |

#### CG 指令

**`@cg show <name> [transition] { }`** — 全屏 CG 展示块

```
@cg show window_stare fade {
  YOU: The city lights blurred through my tears.
}
```

CG 块内可嵌套对话和状态变更。块结束时自动恢复之前的背景和角色。

---

### 4.3 对话

对话指令不用 `@` 前缀，通过行首角色名（全大写）识别。

```
MAURICIO: That's not a library. That's a crime scene.
NARRATOR: Senior year. Day one.
YOU: He hasn't called me that in eight years.
```

- `CHARACTER:` — 角色对白，角色名大小写不敏感（`MAURICIO` = `@mauricio`）
- `NARRATOR:` — 旁白，无角色立绘，独立对话框样式
- `YOU:` — MC 内心独白，斜体或特殊样式

**语法糖：立绘变换 + 对白一行完成**

```
CHARACTER [look]: text
```

等价于：

```
@character look look
CHARACTER: text
```

例：

```
MAURICIO [arms_crossed_angry]: Your call, Butterfly.
```

解释器将其展开为 `CharLookNode`（立绘变换）+ `DialogueNode`（对白），顺序与分开写完全一致。

---

### 4.4 手机/消息

```
@phone show {
  @text from EASTON: Can we talk? I miss you.
  @text from EASTON: I know I messed up.
  @text to MAURICIO: How do you know where I live?
}
@phone hide
```

- `@phone show { }` / `@phone hide`：弹出/收起手机界面
- `@text from <char>: content`：收到的消息（灰色气泡，左对齐）
- `@text to <char>: content`：发出的消息（蓝色气泡，右对齐）

---

### 4.5 音频

```
@music play calm_morning        // 播放 BGM（循环）
@music crossfade tense_strings  // 交叉淡入新 BGM
@music fadeout                  // 淡出停止
@sfx play door_slam             // 一次性音效
```

---

### 4.6 游戏机制

#### `@minigame <game_id> <ATTR> { }`

在叙事流中嵌入小游戏。

```
@minigame qte_challenge ATK {
  @on S {
    NARRATOR: Your reflexes are razor-sharp today.
    @signal mark MINIGAME_PERFECT_EP01
  }
  @on A B {
    NARRATOR: Not bad. You kept up.
  }
  @on C D {
    NARRATOR: Sloppy. You're distracted.
  }
}
```

- `game_id`：小游戏 ID，解释器映射到 OSS URL
- `ATTR`：关联属性（名称由脚本自由定义，解释器不校验具体名称）
- `@on` 支持多评级合并：`@on A B { }`

#### `@choice { }`

选择块，包含多个 `@option`。

```
@choice {
  @option A brave "Stand your ground." {
    check {
      attr: CHA
      dc: 12
    }
    @on success {
      @easton look relieved
      EASTON: Can I sit?
      MALIA: You have two minutes.
      @affection easton +2
      @butterfly "在食堂正面接受了Easton的靠近"
    }
    @on fail {
      @easton look hurt
      MALIA: I... I can't do this.
      @butterfly "试图面对Easton但失去了勇气"
    }
  }
  @option B safe "Have Mark make a scene." {
    @mark show grin_mischief at right
    @easton hide fade
    MARK: HEY EASTON! You want some of my mystery casserole?
    @mark bubble music
    YOU: Thank god for Mark.
    @butterfly "让Mark帮忙避开了和Easton的正面接触"
  }
}
```

#### `@option <ID> <brave|safe> <text> { }`

- `ID`：选项编号（A / B / C...），`@gate` 通过此 ID 引用
- `brave`：需要 D20 检定，必须包含 `check { }` 块和 `@on success/fail` 块
- `safe`：跳过检定，块内直接是叙事内容

#### `check { }`

检定参数块，嵌套在 `@option brave` 内。属性名称不硬编码，脚本可自由使用任何名称（未来属性名变化时无需改语法）。

```
check {
  attr: CHA       // 检定属性（脚本自由定义）
  dc: 12          // 难度值
}
```

D20 检定公式（引擎内置）：`D20(1-20) + 属性修正 + 小游戏修正 >= DC → 成功`

#### `@on <condition> { }`

条件结果块，用于 `@option brave`（success/fail）和 `@minigame`（S/A/B/C/D）。

```
@on success { ... }       // 检定成功
@on fail { ... }          // 检定失败
@on S { ... }             // 小游戏评级 S
@on A B { ... }           // 小游戏评级 A 或 B
```

---

### 4.7 状态变更

脚本只做声明，引擎负责实际计算和持久化。

```
@affection easton +2                  // 好感度
@signal mark EP01_COMPLETE            // 持久布尔标记，可被 @if (EP01_COMPLETE) 查询
@signal achievement HIGH_HEEL_DOUBLE_KILL // 直接触发声明过的成就
@butterfly "正面接受了Easton的靠近"    // 蝴蝶效应记录
```

> **引擎管理的数值**（如 XP、SAN/HP 等）由引擎内部维护，脚本**不能**修改它们。脚本只能在 `@if` 条件中引用这些数值（如 `@if (san <= 20) { }`），具体名称由引擎定义。

#### `@signal <kind> <event>`

事件信号，`kind` 必写，分两类：

| kind | 用途 | 是否可被 `@if` 查询 |
|------|------|-------------------|
| `mark` | **持久布尔标记**。引擎永久存储，在 `@if (NAME)` 条件中作为布尔值使用。触发隐藏剧情线、成就判定、数值比较。 | 是（作为 `FlagCondition`） |
| `achievement` | **成就触发事件**。直接引用 `@achievement <id>` 声明过的成就 ID。引擎收到后走成就系统（UI 弹窗、成就页、数据分析）。 | 否——成就是**外发通知**，不是剧情状态 |

`event` 可为裸标识符或双引号字符串。示例：

```
@signal mark MINIGAME_PERFECT_EP01         // 打完完美评级，打一个持久标记
@signal mark "EP01_COMPLETE"               // 字符串形式等价
@signal achievement HIGH_HEEL_WARRIOR      // 玩家触发成就，由后端推弹窗
&signal mark EASTON_LET_APPROACH           // 并发形式也支持
```

**旧写法 `@signal EVENT`（无 kind）已不再被接受**——必须显式标注 `mark` 或 `achievement`。

#### `@butterfly <description>`

引擎累积，供 `@gate` 中 influence 条件的 LLM 判定。

#### `@achievement <id> { ... }`

**成就声明块**，挂在 episode 顶层（可与 `@gate`/`@ending` 并存）。字段对齐 `cdotlock/story-achievement-generator` skill 的输出 schema——包含 name / rarity / description / when 四个必填字段：

```
@achievement HIGH_HEEL_DOUBLE_KILL {
  name: "【高跟鞋双杀】"
  rarity: epic
  description: "用高跟鞋当武器，一次是即兴，两次是签名招式。"
  when: (HIGH_HEEL_EP05 && HIGH_HEEL_EP24)
}
```

字段说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | 字符串 | 显示名称。推荐 `【……】` 包围、≤8 个汉字 |
| `rarity` | 标识符 | `uncommon` / `rare` / `epic` / `legendary`（**无 `common`**——成就必须需要有意识的玩家行为） |
| `description` | 字符串 | DM 口吻的 1-2 句 flavor 文本 |
| `when` | 条件表达式 | 触发条件 AST，语法与 `@if` 完全一致。通常引用一个或多个 `mark`（组合条件用于 arc 成就） |

**工作机制**：
1. 剧本用 `@achievement` 声明成就，定义其 `when` 触发条件
2. 剧本用 `@signal mark NAME` 打下标记，作为玩家行为的凭据
3. `when` 条件被满足后，引擎解锁成就并推送通知
4. 也可以用 `@signal achievement <id>` **直接**触发（跳过 `when` 判定），用于脚本手动控制触发时机

**ID 约束**：每个 episode 内 `@achievement` ID 不可重复。跨集引用由上层剧本管线保证唯一。

**与成就生成 skill 的关系**：`@achievement` 块的字段结构镜像了 `cdotlock/story-achievement-generator` 生成的 JSON。成就生成器产出成就表后，剧作者（或 LLM 管线）将其落为 MSS 的 `@achievement` 块，加上对应的 `@signal mark` 打点即可。

---

### 4.8 流程控制

#### `@if (<condition>) { }` / `@else @if (<condition>) { }` / `@else { }`

条件分支。条件必须用括号 `()` 包裹。支持 `&&`（与）和 `||`（或）。支持 `@else @if` 链式判定。

```
@if (affection.easton >= 5 && CHA >= 14) {
  EASTON: You remembered.
} @else @if (affection.easton >= 3) {
  EASTON: ...I wasn't sure you'd come.
} @else {
  EASTON: ...Hey.
}

@if (san <= 20 || FAILED_TWICE) {
  YOU: I can barely keep it together.
}
```

**条件类型（5 种，body `@if` 和 gate `@if` 均可使用）。所有类型都解析为结构化 AST，后端直接消费：**

| 类型 | 源语法 | AST 输出 |
|------|------|---------|
| choice | `<ID>.<result>` | `{type:"choice", option, result}` |
| influence | `influence "desc"` 或 `"desc"` | `{type:"influence", description}` |
| flag | `SIGNAL_NAME` | `{type:"flag", name}` |
| comparison | `affection.<char> <op> <N>` 或 `<name> <op> <N>` | `{type:"comparison", left:{kind,...}, op, right}` |
| compound | `expr && expr` / `expr \|\| expr` | `{type:"compound", op, left, right}`（递归嵌套） |

**操作符：** `>=` `<=` `>` `<` `==` `!=`

**结构化比较 AST**：左侧 `left.kind` 为 `"affection"`（携带 `char` 字段）或 `"value"`（携带 `name` 字段）；右侧 `right` 恒为整数。

**复合条件 AST**：`left` 和 `right` 是递归的完整条件 JSON 对象，不是字符串。

---

### 4.9 并发控制

MSS 用 `@` 和 `&` 两种前缀区分指令的时序关系。

#### 时序语义

| 前缀 | 含义 | 角色 |
|------|------|------|
| `@` | 顺序执行，开启新的步骤组 | **领导者**（leader） |
| `&` | 并发执行，加入前一个步骤组 | **跟随者**（follower） |
| （无前缀） | 对话行，始终独立——等待玩家点击 | **独立** |

#### 工作原理

`@` 指令开启一个新的步骤组。紧随其后的 `&` 指令加入同一组，与 `@` 指令同时执行。遇到下一条 `@` 指令或对话行时，当前组关闭。

```
// 场景搭建——三条指令并发执行
@bg set school_hallway fade
&music crossfade tense_strings
&mauricio show neutral_smirk at right

// 对话——每行独立，等待玩家点击
NARRATOR: The hallway fell silent.
MAURICIO: Hey, Butterfly.
```

上例中 `@bg`、`&music`、`&mauricio` 构成一个并发组，引擎同时执行三条指令（切背景 + 换音乐 + 角色入场）。两条对话行各自独立，逐条推进。

#### 适用范围

`&` 可用于所有**简单指令**（背景、角色、音频、状态变更等）。

**不可用于块结构指令：**
- `@choice { }` — 选择块
- `@cg show { }` — CG 块
- `@minigame { }` — 小游戏块
- `@phone show { }` — 手机界面块
- `@if { }` — 条件分支块
- `@gate { }` — 路由声明块

块结构指令必须用 `@` 前缀独立执行。

#### 典型用法

**场景切换**——背景、音乐、角色同步切换：

```
@bg set school_cafeteria fade
&music crossfade casual_lunch
&mark show grin_confident at right
&malia show neutral_flat at left
```

**角色同时动作**——多角色同时变换：

```
@mauricio look arms_crossed_angry
&josie bubble sweat
```

---

## 5. 素材映射

### 5.1 设计原则

脚本和素材映射**分离**：
- 脚本只写语义名（如 `malias_bedroom_morning`、`neutral_smirk`）
- 素材映射表是独立文件，由素材管线（Agent-Forge）生成维护
- 解释器将两者结合：`mss compile script.md --assets mapping.json -o output.json`

分离的好处：
- 同一套脚本可指向不同 OSS 环境（开发/生产/测试）
- 素材更新（换 URL、换 CDN）不需要改脚本
- 素材管线和剧本管线可以独立迭代

### 5.2 映射表格式

```json
{
  "base_url": "https://oss.mobai.com/novel_001",
  "assets": {
    "bg": {
      "malias_bedroom_morning": "bg/malias_bedroom_morning.png",
      "malias_bedroom_night": "bg/malias_bedroom_night.png",
      "school_front": "bg/school_front.png",
      "school_hallway": "bg/school_hallway.png",
      "school_classroom": "bg/school_classroom.png",
      "school_cafeteria": "bg/school_cafeteria.png",
      "school_gymnasium": "bg/school_gymnasium.png"
    },
    "characters": {
      "mauricio": {
        "neutral_smirk": "characters/mauricio_neutral_smirk.png",
        "arms_crossed_angry": "characters/mauricio_arms_crossed_angry.png"
      },
      "malia": {
        "neutral_phone": "characters/malia_neutral_phone.png",
        "neutral_flat": "characters/malia_neutral_flat.png",
        "worried": "characters/malia_worried.png",
        "shocked": "characters/malia_shocked.png"
      },
      "easton": {
        "vulnerable_hopeful": "characters/easton_vulnerable_hopeful.png",
        "relieved": "characters/easton_relieved.png",
        "hurt": "characters/easton_hurt.png"
      }
    },
    "music": {
      "calm_morning": "music/calm_morning.mp3",
      "upbeat_school": "music/upbeat_school.mp3",
      "tense_strings": "music/tense_strings.mp3"
    },
    "sfx": {
      "crowd_noise": "sfx/crowd_noise.mp3",
      "phone_buzz": "sfx/phone_buzz.mp3"
    },
    "cg": {
      "window_stare": "cg/window_stare.png"
    },
    "minigames": {
      "qte_challenge": "minigames/qte_challenge/index.html"
    }
  }
}
```

### 5.3 解析规则

解释器按以下规则映射语义名到 URL：

| 脚本指令 | 映射路径 | 完整 URL |
|---------|---------|---------|
| `@bg set malias_bedroom_morning` | `assets.bg.malias_bedroom_morning` | `{base_url}/bg/malias_bedroom_morning.png` |
| `@mauricio show neutral_smirk at right` | `assets.characters.mauricio.neutral_smirk` | `{base_url}/characters/mauricio_neutral_smirk.png` |
| `@music play calm_morning` | `assets.music.calm_morning` | `{base_url}/music/calm_morning.mp3` |
| `@sfx play crowd_noise` | `assets.sfx.crowd_noise` | `{base_url}/sfx/crowd_noise.mp3` |
| `@cg show window_stare` | `assets.cg.window_stare` | `{base_url}/cg/window_stare.png` |
| `@minigame qte_challenge` | `assets.minigames.qte_challenge` | `{base_url}/minigames/qte_challenge/index.html` |

映射表中找不到的语义名 → 解释器报错（validate 模式下）或输出 warning + 空 URL（compile 模式下）。

---

## 6. 解释器

### 6.1 概述

Go 单二进制工具 `mss`。

```bash
mss compile 01.md --assets mapping.json -o ep01.json        # 单集编译
mss compile main/ --assets mapping.json -o novel.json        # 批量编译整个目录
mss validate 01.md --assets mapping.json                     # 验证（不输出 JSON）
```

### 6.2 核心职责

1. **解析**：将自定义语法解析为 AST
2. **验证**：语法正确性、引用完整性（角色是否 show 过才能 look、label 是否存在等）
3. **素材映射**：通过映射表将语义名翻译为完整 OSS URL
4. **输出 JSON**：前端播放器可直接消费的结构化数据

### 6.3 JSON 输出结构

设计原则：**类型明确、URL 已解析、混合数组表达并发**。

`steps` 数组包含两种元素：
- **对象**（object）：单个步骤，顺序执行
- **数组**（array）：并发组，组内所有步骤同时执行

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
        "url": "https://oss.mobai.com/novel_001/bg/malias_bedroom_morning.png",
        "transition": "dissolve"
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
        {"direction": "from", "character": "easton", "text": "Can we talk? I miss you."},
        {"direction": "from", "character": "easton", "text": "I know I messed up."}
      ]
    },
    {
      "type": "phone_hide"
    },
    {
      "type": "pause",
      "clicks": 1
    },
    {
      "type": "minigame",
      "game_id": "qte_challenge",
      "game_url": "https://oss.mobai.com/novel_001/minigames/qte_challenge/index.html",
      "attr": "ATK",
      "on_results": {
        "S": [
          {"type": "narrator", "text": "Your reflexes are razor-sharp today."},
          {"type": "signal", "kind": "mark", "event": "MINIGAME_PERFECT_EP01"}
        ],
        "A B": [
          {"type": "narrator", "text": "Not bad. You kept up."}
        ],
        "C D": [
          {"type": "narrator", "text": "Sloppy. You're distracted."}
        ]
      }
    },
    {
      "type": "choice",
      "options": [
        {
          "id": "A",
          "mode": "brave",
          "text": "Stand your ground.",
          "check": {
            "attr": "CHA",
            "dc": 12
          },
          "on_success": [
            {"type": "char_look", "character": "easton", "look": "relieved", "url": "https://oss.mobai.com/..."},
            {"type": "dialogue", "character": "easton", "text": "Can I sit?"},
            {"type": "dialogue", "character": "malia", "text": "You have two minutes."},
            {"type": "affection", "character": "easton", "delta": 2},
            {"type": "butterfly", "description": "在食堂正面接受了Easton的靠近"}
          ],
          "on_fail": [
            {"type": "char_look", "character": "easton", "look": "hurt", "url": "https://oss.mobai.com/..."},
            {"type": "dialogue", "character": "malia", "text": "I... I can't do this."},
            {"type": "butterfly", "description": "试图面对Easton但失去了勇气"}
          ]
        },
        {
          "id": "B",
          "mode": "safe",
          "text": "Have Mark make a scene.",
          "steps": [
            {"type": "char_show", "character": "mark", "look": "grin_mischief", "url": "https://oss.mobai.com/...", "position": "right"},
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
      "else": {
        "type": "if",
        "condition": {"type": "comparison", "left": {"kind": "affection", "char": "easton"}, "op": ">=", "right": 3},
        "then": [
          {"type": "dialogue", "character": "easton", "text": "...I wasn't sure you'd come."}
        ],
        "else": [
          {"type": "dialogue", "character": "easton", "text": "...Hey."}
        ]
      }
    }
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
  "ending": null
}
```

**JSON 设计原则：**
- 所有 `character` 字段在 JSON 输出中统一为**小写**（如脚本中 `MAURICIO:` → JSON 中 `"character": "mauricio"`）
- 每个 step 有 `type` 字段做类型判别，前端用 switch/case 消费
- URL 全部已解析，前端不需要二次拼接
- `steps` 是混合数组：对象 = 单步骤，数组 = 并发组（同时执行）
- 单步骤的并发组（只有一个 `@` 无 `&` 跟随）自动展平为对象，不包裹数组
- 嵌套只出现在有分支的地方（choice.options、minigame.on_results、if.then/else、gate if/else）
- `gate` 采用嵌套 if/else 链而非平坦数组，条件为**完全结构化的 AST**（不含任何表达式字符串，后端可直接消费）
- `gate` 与 `ending` 两个字段**恒存在**：路由集 `ending` 为 `null`，终结集 `gate` 为 `null`
- `ending` 为 `{type: "complete" | "to_be_continued" | "bad_ending"}`，同时出现 `gate` 和 `ending` 是编译错误
- safe 选项的内容在 `steps` 字段，brave 选项的内容在 `on_success`/`on_fail` 字段

---

## 7. Remix 兼容

### 7.1 Remix 输出格式

Remix Executor 输出标准 `.md` 文件，与 Dramatizer 产出完全一致。

### 7.2 两种生命周期

**完整集替换**：Remix 生成一集或多集完整内容，放在 `remix/<session_id>/` 目录下。

**选择后片段**：用户输入替换了原 `@choice`，Remix 生成从选择点之后的全部内容，作为完整 `@episode` 输出。

### 7.3 示例

```
@episode remix/abc123:01 "Mark's Casserole Incident" {

  @mark show grin_mischief at right
  MARK: FOOD FIGHT!
  @sfx play crowd_noise

  NARRATOR: The cafeteria descended into chaos.
  YOU: This is either the best or worst idea Mark has ever had.

  @butterfly "Mark 发起了食堂食物大战"

  @gate {
    // 回归主线第二集
    @else: @next main:02
  }
}
```

### 7.4 回归机制

- **回归主线**：`@else: @next main:02`
- **不回归**：`@else: @next remix/abc123:02`（指向自己的下一集）
- **条件回归**：用 `@if` 条件定义

---

## 附录 A：Episode 1 完整脚本

```
@episode main:01 "Butterfly" {

  // ===== 场景一：Malia 卧室，清晨 =====

  @bg set malias_bedroom_morning
  &music play calm_morning
  &malia show neutral_phone at left

  NARRATOR: Senior year. Day one. Status: already complicated.
  YOU: Another year. Same mess.

  @phone show {
    @text from EASTON: Can we talk? I miss you.
    @text from EASTON: I know I messed up.
  }
  @phone hide

  YOU: Eight months and he still won't stop.
  @malia look worried

  // ===== 场景二：学校门口 =====

  @bg set school_front fade
  &music crossfade upbeat_school
  &malia show neutral_flat at left
  &josie show cheerful_wave at right

  JOSIE: Malia! Over here!
  @josie bubble heart
  MALIA: Hey, Jo.
  JOSIE: New year, new Malia. That's the plan, right?
  YOU: If only it were that simple.

  // ===== 场景三：走廊 =====

  @bg set school_hallway fade
  &josie look nervous_whisper
  JOSIE: Don't look. Three o'clock.

  @mauricio show neutral_smirk at right
  @malia hide

  MAURICIO: Hey, Butterfly.
  @josie bubble sweat
  YOU: He hasn't called me that in eight years.

  // ===== 场景四：教室 + 小游戏 =====

  @bg set school_classroom fade
  &music crossfade tense_strings
  &malia show neutral_flat at left

  NARRATOR: Mr. Chen paired them together. Of course he did.

  @minigame qte_challenge ATK {
    @on S {
      NARRATOR: Your reflexes are razor-sharp today.
      @signal mark MINIGAME_PERFECT_EP01
    }
    @on A B {
      NARRATOR: Not bad. You kept up.
    }
    @on C D {
      NARRATOR: Sloppy. You're distracted.
    }
  }

  // ===== 场景五：食堂 + 核心选择 =====

  @bg set school_cafeteria fade
  &music crossfade casual_lunch
  &mark show grin_confident at right
  &malia show neutral_flat at left

  MARK: Yo, Malia! Sit with us!
  @sfx play crowd_noise

  @easton show vulnerable_hopeful at right
  @mark hide

  NARRATOR: Easton is walking toward your table.

  @choice {
    @option A brave "Let him come." {
      check {
        attr: CHA
        dc: 12
      }
      @on success {
        @easton look relieved
        EASTON: Can I sit?
        MALIA: You have two minutes.
        @affection easton +2
        @butterfly "在食堂正面接受了Easton的靠近"
      }
      @on fail {
        @easton look hurt
        MALIA: I... I can't do this.
        EASTON: Malia, please—
        MALIA: Not here. Not now.
        @butterfly "试图面对Easton但失去了勇气"
      }
    }
    @option B safe "Have Mark make a scene." {
      @mark show grin_mischief at right
      @easton hide fade
      MARK: HEY EASTON! You want some of my mystery casserole?
      @mark bubble music
      YOU: Thank god for Mark.
      @butterfly "让Mark帮忙避开了和Easton的正面接触"
    }
  }

  // ===== 场景六：体育馆 =====

  @bg set school_gymnasium fade
  &music crossfade ambient_gym
  &malia show neutral_flat at left
  &josie show excited at right

  JOSIE: Did you see Elias in practice today?
  YOU: I was trying not to.

  @elias show neutral_calm at right
  @josie hide

  NARRATOR: He didn't say a word. He didn't have to.
  @malia bubble heart

  // ===== 场景七：夜晚卧室 =====

  @bg set malias_bedroom_night fade
  &music crossfade night_piano
  &malia show neutral_phone at left

  YOU: Day one. Survived. Barely.

  @phone show {
    @text from UNKNOWN: nice curtains, Butterfly
  }
  @phone hide

  @malia look shocked
  YOU: ...How does he know which window is mine?

  @signal mark EP01_COMPLETE

  // ===== 路由 =====

  @gate {
    @if (A.fail): @next main/bad/001:01
    @else @if ("玩家展现过对Easton的持续接纳"): @next main/route/001:01
    @else: @next main:02
  }
}
```

---

## 附录 B：指令速查表

| 指令 | 说明 |
|------|------|
| `@episode <branch_key> <title> { }` | 集定义 |
| `@gate { }` | 路由声明（集尾部，与 `@ending` 二选一） |
| `@ending <type>` | 终结标记（`complete` / `to_be_continued` / `bad_ending`，与 `@gate` 二选一） |
| `@if (<condition>): @next <branch_key>` | 跳转规则（括号必需，如 `@if (A.fail): @next ...`） |
| `@else @if (<condition>): @next <branch_key>` | 链式跳转规则 |
| `@else: @next <branch_key>` | 兜底路线 |
| `@label <name>` | 跳转锚点 |
| `@goto <name>` | 跳转（高级，慎用） |
| `@pause for <N>` | 等待 N 次点击后继续 |
| `@<char> show <look> at <pos>` | 角色入场 |
| `@<char> hide [transition]` | 角色退场 |
| `@<char> look <look> [transition]` | 换立绘 |
| `@<char> move to <pos>` | 角色移位 |
| `@<char> bubble <type>` | 气泡动画 |
| `@bg set <name> [transition]` | 切背景 |
| `@cg show <name> [transition] { }` | CG 展示块 |
| `CHARACTER: text` | 对白 |
| `NARRATOR: text` | 旁白 |
| `YOU: text` | 内心独白 |
| `@phone show { }` / `@phone hide` | 手机界面 |
| `@text from <char>: content` | 收到消息 |
| `@text to <char>: content` | 发出消息 |
| `@music play <name>` | 播放 BGM |
| `@music crossfade <name>` | 交叉淡入新 BGM |
| `@music fadeout` | 淡出停止 |
| `@sfx play <name>` | 播放音效 |
| `@minigame <game_id> <ATTR> { }` | 小游戏 |
| `@choice { }` | 选择块 |
| `@option <ID> <brave\|safe> <text> { }` | 选项（带编号） |
| `check { attr / dc }` | 检定参数（嵌套在 brave 选项内） |
| `@on <condition> { }` | 条件结果块 |
| `@affection <char> <+/-N>` | 好感度 |
| `@signal <kind> <event>` | 事件信号。kind: `mark`（持久布尔标记，可在 `@if` 中检查）/ `achievement`（触发声明过的成就） |
| `@achievement <id> { name / rarity / description / when }` | 成就声明（字段对齐 story-achievement-generator skill） |
| `@butterfly <description>` | 蝴蝶效应 |
| `@if (<condition>) { }` / `@else @if (<condition>) { }` / `@else { }` | 条件分支（括号必需，支持链式 @else @if） |
| `&<指令>` | 并发前缀，与前一条 `@` 同时执行（不可用于块结构） |

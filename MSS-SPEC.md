# MoonShort Script (MSS) 格式设计规范 v2.1

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
| 状态变更 | XP、SAN、好感度、信号、蝴蝶效应 |
| 流程控制 | 条件分支、标签跳转 |

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

路由声明块，**必须位于 `@episode` 块的尾部**。引擎按声明顺序从上到下判定条件，第一个命中的 `@if` 生效；全不命中则走 `@else`（或裸 `@next`）。

```
@gate {
  @if (A fail): @next main/bad/001:01
  @if ("玩家展现过对Easton的持续接纳"): @next main/route/001:01
  @else: @next main:02
}
```

#### `@if (<condition>): @next <branch_key>`

单条跳转规则。两种条件格式：

**格式一：choice（直接根据选择跳转）**
```
@if (A fail): @next main/bad/001:01
```

条件格式：`<option_id> <check_result>`
- option_id：选项编号（A / B / C 等，对应 `@option` 的 ID）
- check_result：`success` | `fail` | `any`

**格式二：influence（LLM 检定蝴蝶效应）**
```
@if ("玩家多次展现对弱者的同情心"): @next main/route/001:01
```

运行时，引擎将玩家积累的所有 `@butterfly` 记录喂给 LLM，判断是否满足条件。

#### `@else: @next <branch_key>`

兜底路线，所有 `@if` 都不命中时走这里。

#### `@label <name>` / `@goto <name>`

集内跳转锚点。**高级指令，慎用**。

```
@if SKIP_FIGHT {
  @goto AFTER_FIGHT
}

// ... 打斗内容 ...

@label AFTER_FIGHT
NARRATOR: The dust settled.
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
    @signal MINIGAME_PERFECT_EP01
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
      @xp +3
      @san +5
      @butterfly "在食堂正面接受了Easton的靠近"
    }
    @on fail {
      @easton look hurt
      MALIA: I... I can't do this.
      @san -20
      @xp +1
      @butterfly "试图面对Easton但失去了勇气"
    }
  }
  @option B safe "Have Mark make a scene." {
    @mark show grin_mischief at right
    @easton hide fade
    MARK: HEY EASTON! You want some of my mystery casserole?
    @mark bubble music
    YOU: Thank god for Mark.
    @xp +1
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
@xp +3                    // 经验值
@san -20                   // 理智值
@affection easton +2       // 好感度
@signal EP01_COMPLETE      // 向引擎抛出事件
@butterfly "正面接受了Easton的靠近"  // 蝴蝶效应记录
```

- `@signal <event>`：引擎决定如何处理（存标记、触发成就、解锁路线等）
- `@butterfly <description>`：引擎累积，供 `@gate` 中 influence 条件的 LLM 判定

---

### 4.8 流程控制

#### `@if <condition> { }` / `@else { }`

条件分支。支持 `&&`（与）和 `||`（或）。

```
@if affection.easton >= 5 && CHA >= 14 {
  EASTON: You remembered.
} @else {
  EASTON: ...Hey.
}

@if san <= 20 || FAILED_TWICE {
  YOU: I can barely keep it together.
}
```

**判定对象：**

| 对象 | 语法 | 说明 |
|------|------|------|
| flag | `FLAG_NAME` | 布尔，该信号是否被触发 |
| 好感度 | `affection.<char> <op> <N>` | 角色好感度数值 |
| 属性 | `<ATTR_NAME> <op> <N>` | 玩家属性值（名称自由） |
| 经验 | `xp <op> <N>` | 当前经验值 |
| 理智 | `san <op> <N>` | 当前理智值 |

**操作符：** `>=` `<=` `>` `<` `==` `!=`

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

设计原则：**类型明确、URL 已解析、扁平易遍历**。

```json
{
  "episode_id": "main:01",
  "branch_key": "main",
  "seq": 1,
  "title": "Butterfly",
  "steps": [
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
    },
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
      "type": "minigame",
      "game_id": "qte_challenge",
      "game_url": "https://oss.mobai.com/novel_001/minigames/qte_challenge/index.html",
      "attr": "ATK",
      "on_results": {
        "S": [
          {"type": "narrator", "text": "Your reflexes are razor-sharp today."},
          {"type": "signal", "event": "MINIGAME_PERFECT_EP01"}
        ],
        "A,B": [
          {"type": "narrator", "text": "Not bad. You kept up."}
        ],
        "C,D": [
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
            {"type": "xp", "delta": 3},
            {"type": "san", "delta": 5},
            {"type": "butterfly", "description": "在食堂正面接受了Easton的靠近"}
          ],
          "on_fail": [
            {"type": "char_look", "character": "easton", "look": "hurt", "url": "https://oss.mobai.com/..."},
            {"type": "dialogue", "character": "malia", "text": "I... I can't do this."},
            {"type": "san", "delta": -20},
            {"type": "xp", "delta": 1},
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
            {"type": "xp", "delta": 1},
            {"type": "butterfly", "description": "让Mark帮忙避开了和Easton的正面接触"}
          ]
        }
      ]
    },
    {
      "type": "if",
      "condition": "affection.easton >= 5 && CHA >= 14",
      "then": [
        {"type": "dialogue", "character": "easton", "text": "You remembered."}
      ],
      "else": [
        {"type": "dialogue", "character": "easton", "text": "...Hey."}
      ]
    }
  ],
  "gate": [
    {
      "condition": "A fail",
      "next": "main/bad/001:01"
    },
    {
      "condition": "玩家展现过对Easton的持续接纳",
      "next": "main/route/001:01"
    },
    {
      "next": "main:02"
    }
  ]
}
```

**JSON 设计原则：**
- 每个 step 有 `type` 字段做类型判别，前端用 switch/case 消费
- URL 全部已解析，前端不需要二次拼接
- `steps` 是扁平数组，顺序即执行顺序
- 嵌套只出现在有分支的地方（choice.options、minigame.on_results、if.then/else）
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
  @music play calm_morning
  @malia show neutral_phone at left

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
  @music crossfade upbeat_school
  @malia show neutral_flat at left
  @josie show cheerful_wave at right

  JOSIE: Malia! Over here!
  @josie bubble heart
  MALIA: Hey, Jo.
  JOSIE: New year, new Malia. That's the plan, right?
  YOU: If only it were that simple.

  // ===== 场景三：走廊 =====

  @bg set school_hallway fade
  @josie look nervous_whisper
  JOSIE: Don't look. Three o'clock.

  @mauricio show neutral_smirk at right
  @malia hide

  MAURICIO: Hey, Butterfly.
  @josie bubble sweat
  YOU: He hasn't called me that in eight years.

  // ===== 场景四：教室 + 小游戏 =====

  @bg set school_classroom fade
  @music crossfade tense_strings
  @malia show neutral_flat at left

  NARRATOR: Mr. Chen paired them together. Of course he did.

  @minigame qte_challenge ATK {
    @on S {
      NARRATOR: Your reflexes are razor-sharp today.
      @signal MINIGAME_PERFECT_EP01
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
  @music crossfade casual_lunch
  @mark show grin_confident at right
  @malia show neutral_flat at left

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
        @xp +3
        @san +5
        @butterfly "在食堂正面接受了Easton的靠近"
      }
      @on fail {
        @easton look hurt
        MALIA: I... I can't do this.
        EASTON: Malia, please—
        MALIA: Not here. Not now.
        @san -20
        @xp +1
        @butterfly "试图面对Easton但失去了勇气"
      }
    }
    @option B safe "Have Mark make a scene." {
      @mark show grin_mischief at right
      @easton hide fade
      MARK: HEY EASTON! You want some of my mystery casserole?
      @mark bubble music
      YOU: Thank god for Mark.
      @xp +1
      @butterfly "让Mark帮忙避开了和Easton的正面接触"
    }
  }

  // ===== 场景六：体育馆 =====

  @bg set school_gymnasium fade
  @music crossfade ambient_gym
  @malia show neutral_flat at left
  @josie show excited at right

  JOSIE: Did you see Elias in practice today?
  YOU: I was trying not to.

  @elias show neutral_calm at right
  @josie hide

  NARRATOR: He didn't say a word. He didn't have to.
  @malia bubble heart

  // ===== 场景七：夜晚卧室 =====

  @bg set malias_bedroom_night fade
  @music crossfade night_piano
  @malia show neutral_phone at left

  YOU: Day one. Survived. Barely.

  @phone show {
    @text from UNKNOWN: nice curtains, Butterfly
  }
  @phone hide

  @malia look shocked
  YOU: ...How does he know which window is mine?

  @signal EP01_COMPLETE

  // ===== 路由 =====

  @gate {
    @if (A fail): @next main/bad/001:01
    @if ("玩家展现过对Easton的持续接纳"): @next main/route/001:01
    @else: @next main:02
  }
}
```

---

## 附录 B：指令速查表

| 指令 | 说明 |
|------|------|
| `@episode <branch_key> <title> { }` | 集定义 |
| `@gate { }` | 路由声明（集尾部） |
| `@if (<condition>): @next <branch_key>` | 跳转规则 |
| `@else: @next <branch_key>` | 兜底路线 |
| `@label <name>` | 跳转锚点 |
| `@goto <name>` | 跳转（高级，慎用） |
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
| `@xp <+/-N>` | 经验值 |
| `@san <+/-N>` | 理智值 |
| `@affection <char> <+/-N>` | 好感度 |
| `@signal <event>` | 抛出事件 |
| `@butterfly <description>` | 蝴蝶效应 |
| `@if <condition> { }` / `@else { }` | 条件分支 |

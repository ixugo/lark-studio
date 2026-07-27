# 术语指定翻译

vdub 支持用户自定义术语映射表，在翻译时强制将指定词汇翻译为指定内容。

---

## 使用场景

视频翻译中经常遇到不应被翻译或需要特定翻译的词汇：

| 类型 | 示例 | 期望行为 |
|------|------|---------|
| 编程语言 | Golang、Python、Rust | 保持原文 |
| 产品名称 | Kubernetes、Docker | 保持原文 |
| 口语确认词 | OK、Yeah | 保持原文 |
| 人名 | Ardan Labs | 保持原文 |
| 专有术语 | Container | 翻译为"容器" |
| 缩写约定 | API | 翻译为"接口" |

不加术语锁定时，LLM 可能将 "Golang" 翻译为 "Go语言"，
或在不同批次中对同一术语产出不一致的翻译。

---

## 数据模型

```go
type Term struct {
    ID          int64     `gorm:"primaryKey;autoIncrement"`
    Text        string    `gorm:"uniqueIndex;notNull"`     // 源词
    Translation string    `gorm:"notNull"`                  // 译文
    CreatedAt   time.Time
}
```

- `Text` 为源词，唯一索引，不可重复
- `Translation` 为指定译文
- 源词与译文相同时表示"保持原文不翻译"
- 添加时 `Translation` 留空默认等于 `Text`

---

## 匹配规则

### 不区分大小写

```
术语: golang
字幕: "This is written in Golang and deployed on Linux."
匹配: ✓ (Golang ≈ golang)
```

`matchTermMappings` 将术语和字幕全部转为小写后做子串匹配，
"Golang"、"GOLANG"、"golang" 均能命中。

### 子串匹配

在整个批次的字幕文本中搜索术语是否存在：

```go
corpus := strings.ToLower(strings.Join(sentences, " "))
if strings.Contains(corpus, strings.ToLower(term.Text)) {
    // 命中
}
```

### 仅注入命中术语

每个翻译批次（8 句）独立匹配，仅将该批次中实际出现的术语注入 LLM 提示词，
避免大量无关术语稀释翻译指令。

---

## LLM 提示词注入

匹配到的术语以强制指令形式注入翻译提示词尾部：

```
- MANDATORY terminology: for each term below, when you encounter
  it (case-insensitive), you MUST translate it EXACTLY as specified:
  "Golang" → keep as-is; "Container" → "容器"; "API" → "接口"
```

- 源词 = 译文时：`"Golang" → keep as-is`
- 源词 ≠ 译文时：`"Container" → "容器"`

### 为什么用提示词注入而非字符串替换

直接在翻译前/后做字符串替换会产生副作用：

| 方案 | 问题 |
|------|------|
| 翻译前替换为占位符 | LLM 失去上下文，相邻词汇翻译质量下降 |
| 翻译后字符串替换 | 无法处理词形变化、位置偏移 |
| 提示词注入（当前方案）| LLM 理解完整语境后按指令处理，质量最高 |

提示词注入让 LLM 在完整理解上下文的前提下决定如何处理术语，
既保证术语精确，又不影响其他词汇的翻译质量。

---

## API 接口

### 查询所有术语

```
GET /terms

响应:
{
  "data": {
    "items": [
      {"id": 1, "text": "Golang", "translation": "Golang", "created_at": "..."},
      {"id": 2, "text": "Container", "translation": "容器", "created_at": "..."}
    ]
  }
}
```

### 添加术语

```
POST /terms
Content-Type: application/json

{"text": "Golang"}                           → 保持原文
{"text": "Container", "translation": "容器"}  → 指定翻译
```

`text` 必填，`translation` 选填（留空默认等于 `text`）。
`text` 有唯一索引，重复添加返回错误。

### 删除术语

```
DELETE /terms/:id
```

---

## Flutter UI

设置页 → 术语锁定 区域：

```
┌─────────────────────────────────────────────┐
│  术语锁定                                    │
│  翻译时保持原文或指定翻译的专有名词            │
│                                             │
│  [源词 (如 Golang)] → [译文 (留空=保持原文)] [添加] │
│                                             │
│  ┌───────┐ ┌────────────┐ ┌─────────┐      │
│  │Golang ×│ │Container→容器×│ │API→接口 ×│     │
│  └───────┘ └────────────┘ └─────────┘      │
└─────────────────────────────────────────────┘
```

- 输入源词和译文后点击"添加"或按回车
- 已有术语以 Chip 形式展示，点 × 删除
- 源词=译文时仅显示源词，否则显示 `源词 → 译文`

---

## 架构分层

```
API 层                    Core 层                      Store 层
term.go                   term/core.go                 term/store/termdb/db.go
  ├ GET  /terms    →      ListAll()              →     Find all terms
  ├ POST /terms    →      Add(text, translation) →     Create term
  └ DELETE /terms/:id →   Delete(id)             →     Delete by ID

Pipeline 层
  step_translate.go
    ├ injectTermsIntoPrompt()  ← TermLister 接口
    └ matchTermMappings()
```

`pipeline.TermLister` 接口由 `termAdapter` 实现，将 `term.Core` 的
`[]term.Term` 转换为 `[]pipeline.TermMapping`，保持领域间解耦。

---

## 限制与规划

| 限制 | 说明 |
|------|------|
| 无模糊匹配 | 仅精确子串匹配，不支持正则或词干分析 |
| 无词频统计 | 不统计术语在字幕中的出现次数 |
| 全局生效 | 术语表对所有任务生效，不支持按任务自定义 |

未来可扩展方向：
- 按项目/任务绑定术语表
- 导入/导出术语表（CSV/JSON）
- 术语表模板（编程、医学、法律等预设集）

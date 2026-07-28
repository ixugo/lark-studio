# SmartSub → vdub UI 全页面复刻清单

> 参照 SmartSub v3.4.0 (Electron + TypeScript)，逐一复刻到 vdub Flutter Desktop。
> 设计语言：暗色主题为主，Cupertino 风格，支持明/暗切换。

---

## 全局框架

### 1. Layout (全局壳)
**对标**: `renderer/components/Layout.tsx`
**功能**:
- 左侧竖排导航 rail (64px)：Logo + 任务组导航 + 分隔线 + 配置组导航 + 弹性空间 + 设置按钮
- 顶栏 (44px)：当前页面名 + 全局搜索(⌘K) + GPU加速徽章 + 活动中心 + 帮助菜单 + 主题切换
- 底部状态栏 (26px)：引擎就绪 + GPU状态 + 任务运行指示 + 下载进度 + 版本号
- 全局弹窗：更新弹窗、日志弹窗、快捷键速查、FAQ、新手引导、命令面板
**状态**: ✅ 已实现 (64px NavRail + TopBar + StatusBar + 暗色主题)

---

## 任务组 (创作流水线)

### 2. 启动台 (Launchpad / Home)
**对标**: `renderer/pages/[locale]/home.tsx` + `renderer/components/launchpad/`
**功能**:
- 问候语 + 日期 + 任务统计(总数/已完成)
- 工作流卡片网格：配音成片、双语字幕、原文字幕、翻译已有字幕、英文视频转中文、自定义流程
- 每张卡片有彩色图标 + 标题 + 描述，点击创建对应类型任务
- 工具栏：视频下载 / 校对字幕 / 合成到视频 / 配音
- 最近任务列表：状态圆点 + 名称(可重命名) + 类型标签 + 文件数 + 时间 + 状态文字 + 操作按钮
**状态**: ✅ 已实现 (问候语+工作流网格+工具栏+最近任务)

### 3. 下载 (Download)
**对标**: `renderer/pages/[locale]/download.tsx` + `renderer/components/download/`
**功能**:
- 安装下载引擎卡片：yt-dlp (YouTube/1800+sites) / lux (Bilibili/抖音)
- 视频下载面板：粘贴链接文本框(支持批量)、保存目录选择、清晰度/引擎/并发数/字幕下载配置
- 解析链接/直接下载按钮
- Cookie 管理
**状态**: ✅ 已实现 (引擎安装卡片+视频下载面板+选项栏)

### 4. 字幕 (Subtitle / Tasks)
**对标**: `renderer/pages/[locale]/tasks/` + `renderer/components/subtitle/`
**功能**:
- 任务类型路由：generate-translate(双语字幕) / generate(原文字幕) / translate(翻译已有字幕)
- 配置栏：语音模型 + 视频语言 + 翻译成X语言 + 翻译服务 + 输出内容
- 文件导入区：拖拽/点击导入，支持 MP4/MKV/MOV/MP3/WAV 等
- 导入后文件列表/网格视图 + 清空
- 高级选项面板
- 底部：运行日志区 + 开始任务按钮
**状态**: ✅ 已有任务列表页 (文件导入区+配置栏待完善)

### 5. 校对 (Proofread)
**对标**: `renderer/pages/[locale]/proofread.tsx` + `renderer/components/proofread/`
**功能**:
- 字幕编辑器：时间轴 + 原文 + 译文 三列
- 播放器联动
- 单条编辑/删除/合并/拆分
- 快捷键操作
**状态**: ✅ 框架已实现 (文件导入区+样式页面)

### 6. 合成 (SubtitleMerge)
**对标**: `renderer/pages/[locale]/subtitleMerge.tsx`
**功能**:
- 将字幕烧录到视频 / 封装为独立字幕文件
- 视频+字幕文件选择
- 字幕样式配置(字体/大小/颜色/位置)
**状态**: ✅ 框架已实现 (文件导入区页面)

### 7. 配音 (Dubbing)
**对标**: `renderer/pages/[locale]/dubbing.tsx` + `renderer/components/dubbing/`
**功能**:
- TTS 配音 + 声音克隆
- 配音服务选择(OpenAI/Azure/Edge TTS/本地模型)
- 音色选择/试听
- 配音速度/音量调节
- 输出格式配置
**状态**: ✅ 框架已实现 (文件导入区页面)

---

## 配置组

### 8. 引擎 (Engines)
**对标**: `renderer/pages/[locale]/engines.tsx` + `renderer/components/resources/`
**功能**:
- 左侧服务列表：总览 + 本地引擎(whisper.cpp/faster-whisper/FunASR/Qwen3-ASR/FireRedASR/本地命令行) + 云端ASR
- 右侧引擎配置详情：GPU加速方式 + 模型管理(下载/导入/删除/切换) + 检测详情 + 推荐模型
- 模型搜索 + 只看已安装过滤
- 模型路径配置
**状态**: ✅ 已实现 (左右分栏+引擎列表+加速方式+总览)

### 9. 翻译 (Translation)
**对标**: `renderer/pages/[locale]/translation.tsx` + `renderer/components/ProviderForm.tsx`
**功能**:
- 左侧翻译服务列表：总览 + 自定义服务商 + 免费起步(自动免费翻译/必应/谷歌/DeepLX) + AI翻译(OpenAI/Ollama等)
- 仅显示已配置过滤开关
- 右侧配置表单：Base URL + API Key + 模型名称 + 回显对齐校验 + 思考模式 + 测试翻译按钮
- 每种服务商有独立配置项
**状态**: ✅ 已实现 (左右分栏+服务列表+配置表单+测试翻译)

### 10. 词库 (Glossary)  ✅ 已实现基础版
**对标**: `renderer/components/glossary/GlossaryManager.tsx`
**功能**:
- 左侧词库列表(280px)：ON/OFF开关 + 名称 + 词条计数 + 上下排序 + 新增词库
- 右侧详情：词库名+状态标签 + 编辑/导入/导出/删除按钮
- 提示卡片(AI使用规则 + 传统机翻说明)
- 搜索栏 + 词条计数 + 新增词条按钮
- 词条表格(source/target/note/actions) + 虚拟列表
- 空态：CSV 模板下载提示
- 弹窗：新建/编辑词库、新建/编辑词条、删除确认
**状态**: ✅ 已实现 (表格表头+排序+编辑删除+搜索，缺 CSV 导入导出)

### 11. 音色 (TTS Services)
**对标**: `renderer/pages/[locale]/ttsServices.tsx` + `renderer/components/tts/`
**功能**:
- 左侧声音服务列表：+新增 + 本地模型(Kokoro/VITS) + 在线服务(OpenAI/SiliconFlow/local/Edge TTS/Azure/豆包/ElevenLabs)
- 每个服务有状态绿点(可用/不可用)
- 右侧配置：可用标签 + 说明文案 + 音色文档/清除配置/测试连接按钮
- 音色候选列表(标签输入) + 请求超时 + 并发数
**状态**: ✅ 已实现 (左右分栏+服务列表+音色候选+超时并发配置)

### 12. 设置 (Settings)
**对标**: `renderer/pages/[locale]/settings.tsx` + `renderer/components/settings/`
**功能**:
- 系统设置：语言切换 + 启动时检查更新 + 休眠阻止 + 关闭窗口行为
- 存储位置：统一存储目录 + 临时文件目录
- 网络代理：代理模式(不使用/系统/HTTP/SOCKS5) + 地址端口
- 下载源(高级)：镜像配置
- VAD 灵敏度
- 关于：版本/日志
**状态**: ✅ 已实现 (暗色主题对齐+分段配置)

---

## 执行顺序 (按依赖和优先级)

| 序号 | 任务 | 预估行数 | 依赖 |
|------|------|----------|------|
| 1 | Layout 重构（导航rail+顶栏+状态栏） | 400-500 | 无 |
| 2 | 启动台重写（工作流卡片+最近任务） | 400-500 | Layout |
| 3 | 词库补全（编辑词条+排序+表格表头） | 200 | 无 |
| 4 | 翻译服务页 | 400-500 | Layout |
| 5 | 引擎页 | 500-600 | Layout |
| 6 | 音色页 | 400-500 | Layout |
| 7 | 字幕/任务页重构 | 500-600 | Layout |
| 8 | 下载页 | 300-400 | Layout |
| 9 | 配音页 | 300-400 | Layout |
| 10 | 合成页 | 200-300 | Layout |
| 11 | 校对页 | 500-600 | Layout |
| 12 | 设置页对齐 | 200-300 | Layout |

---

## 设计规范

- **色彩**: 暗色主题 bg `#000000`~`#1C1C1E`, 前景 `#FFFFFF`~`#E5E5EA`, 主色 `#007AFF`, 成功 `#34C759`, 警告 `#FF9F0A`, 错误 `#FF3B30`
- **字体**: SF Pro / system-ui, 11-20px 范围
- **间距**: 4px 基础网格, padding 通常 12-24px
- **圆角**: 小组件 6-8px, 卡片 10-14px
- **分隔**: 0.5px border `rgba(255,255,255,0.1)`~`rgba(0,0,0,0.1)`
- **图标**: Lucide 风格(空心描边), 16-20px
- **动效**: 150ms ease-out hover 过渡

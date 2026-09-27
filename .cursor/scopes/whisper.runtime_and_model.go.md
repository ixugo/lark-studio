# Whisper.cpp 跨平台运行时与模型管理架构设计

## 需求目标

VDub 的语音识别（ASR）核心依赖 Whisper.cpp 原生执行环境。目前 Windows 与 macOS 用户在首次使用或切换模型时，面临三大核心痛点：一是国内网络环境下，从官方 GitHub Releases 与 Hugging Face 下载运行时和大体积 GGML 模型（100MB~3GB）经常遇到连接超时、速度极慢甚至无法建立连接；二是缺乏跨平台（Windows .zip 与 macOS .tar.gz）的自动化安装与断点续传管理机制；三是软件目前界面仅支持中文，亟需在侧边栏底部操作区提供中英双语即时切换能力，以满足海内外不同用户群体的多语言使用诉求。

### 问题-后果-方案明细表

| 编号 | 问题场景 | 不解决的后果 | 解决方案 | 状态 |
|------|----------|-------------|---------|------|
| 1 | 国内用户直接请求 Hugging Face 官方地址下载 GGML 模型文件 | 连接被重置或下载速率仅几十 KB/s，用户任务严重受阻 | 自动并发网络竞速探测（`hf-mirror.com` vs `huggingface.co` 等），自动择取延迟最低、连通最优节点，无需用户手动干预 | 实施 |
| 2 | 国内用户直接从 GitHub Release 下载 Whisper.cpp 预编译二进制 | 运行时下载失败，导致无法启动语音识别引擎 | 自动并发探测 GitHub 代理镜像（`ghfast.top`、`ghproxy.net`）与原源，自动切换最优加速通道 | 实施 |
| 3 | Windows 与 macOS 平台归档格式与二进制执行方式差异 | Windows 依赖 `.zip` 及 `whisper-cli.exe`，macOS 依赖 `.tar.gz` 及 `whisper-cli`，解压与路径处理易出错 | 抽象跨平台运行时适配器，支持 Windows `.zip` 与 Unix `.tar.gz` 自动化解包与权限赋予（`0755`） | 实施 |
| 4 | 大模型（如 large-v3 3.1GB）下载途中网络抖动或用户意外退出 | 产生残留残缺文件，再次启动需从头下载，耗费大量时间与带宽 | 实现 HTTP Range 断点续传与 `.tmp` 临时文件保护机制，下载完成后执行原子重命名 | 实施 |
| 5 | 下载资源散乱在系统各处，难于集中治理与清理 | 占用用户不可见目录，卸载或迁移时残存大量数 GB 废弃权重文件 | 所有下载的运行时和模型文件统一收口于用户家目录 `~/.lark-studio` 下集中管理 | 实施 |
| 6 | 界面目前为纯中文，缺少英文国际化支持 | 海外用户及习惯英文界面的用户无法顺畅使用 | 构建轻量类型安全的前端 i18n 系统，在侧边栏底部主题切换旁提供一键中英文切换 | 实施 |

---

## 跨平台技术矩阵对比表

| 对比维度 | macOS 平台 | Windows 平台 |
|---|---|---|
| **二进制名称** | `whisper-cli` | `whisper-cli.exe` |
| **归档分发格式** | `tar.gz` | `zip` / `tar.gz` |
| **硬件加速框架** | Apple Silicon Metal (GPU) 自动开启 | AVX2 / CPU / DirectML (可拓展) |
| **默认存储目录** | `~/.lark-studio/models/` | `~/.lark-studio/models/` |
| **解包处理逻辑** | `archive/tar` + `compress/gzip`，恢复可执行权限 `0755` | `archive/zip`，提取至目标运行时目录并生成绝对路径 |
| **探针执行命令** | `whisper-cli -m __probe__ -f /dev/null` | `whisper-cli.exe -m __probe__ -f NUL` |

---

## 主流程图 (flowchart TD)

```mermaid
flowchart TD
    Start([用户进入语音识别引擎页面]) --> CheckRuntime{检测 Whisper 运行时}

    %% 运行时检测与安装分支
    CheckRuntime -->|未安装| PromptInstall[提示安装 Whisper.cpp 运行时]
    PromptInstall --> SelectRuntimeMirror{选择下载源}
    SelectRuntimeMirror -->|国内加速| SetGHProxy[应用国内加速代理节点 ghfast.top]
    SelectRuntimeMirror -->|官方原生| SetGHOrigin[连接 GitHub Release 官方源]
    SetGHProxy --> StartRuntimeDownload[下载平台专属归档包 .zip 或 .tar.gz]
    SetGHOrigin --> StartRuntimeDownload
    StartRuntimeDownload --> ExtractArchive[解压归档并校验 whisper-cli 权限]
    ExtractArchive --> RuntimeInstalled[运行时就绪]

    CheckRuntime -->|已安装| RuntimeInstalled

    %% 模型选择与管理分支
    RuntimeInstalled --> LoadModelList[加载并呈现 GGML 模型列表]
    LoadModelList --> UserAction{用户操作模型}

    UserAction -->|切换当前使用模型| CheckModelExist{本地是否存在}
    CheckModelExist -->|存在| SetActiveModel[更新 config.pipeline.whisper_model]
    SetActiveModel --> HotReload[即时热生效并通知前端]

    CheckModelExist -->|不存在| PromptDownloadModel[提示先下载模型]

    UserAction -->|点击下载模型| CheckDownloading{是否正在下载}
    CheckDownloading -->|是| RejectDuplicate[忽略并提示正在下载中]
    CheckDownloading -->|否| SelectModelMirror{选择模型镜像源}

    SelectModelMirror -->|国内镜像推荐| UseHFMirror[构建 hf-mirror.com 镜像端点]
    SelectModelMirror -->|官方镜像| UseHFOrigin[构建 huggingface.co 官方端点]

    UseHFMirror --> InitDownloadTask[初始化下载任务与状态机]
    UseHFOrigin --> InitDownloadTask

    InitDownloadTask --> CheckResume{检查 .tmp 文件与 Range 支持}
    CheckResume -->|支持断点续传| SetRangeHeader[注入 HTTP Header: Range bytes=offset-]
    CheckResume -->|全新下载| SetNormalHeader[标准 HTTP GET 请求]

    SetRangeHeader --> StreamDownload[流式写入临时文件 .tmp]
    SetNormalHeader --> StreamDownload

    StreamDownload --> ThrottleNotify[每秒/每百分比向前端广播进度与瞬时速度]
    ThrottleNotify --> CompleteDownload{数据流写入完毕?}

    CompleteDownload -->|网络中断/手动取消| SaveTmpKeepState[保留断点 .tmp 记录错误状态]
    CompleteDownload -->|成功完成| VerifyFileSize[校验文件大小合法性]

    VerifyFileSize -->|校验不一致| DeleteCorrupted[删除残损文件并报错]
    VerifyFileSize -->|校验成功| AtomicRename[原子重命名 .tmp 为 .bin]

    AtomicRename --> UpdateModelStatus[标记为已下载并广播 model_download_done]
    UpdateModelStatus --> SuggestActive[提供一键设为当前模型]

    UserAction -->|删除已有模型| ConfirmDelete{二次确认?}
    ConfirmDelete -->|确认| RemoveModelFile[从磁盘安全移除模型文件]
    RemoveModelFile --> RefreshList[刷新本地模型列表]
```

---

## 核心时序图 (sequenceDiagram)

```mermaid
sequenceDiagram
    autonumber
    actor User as 用户界面 (React)
    participant i18n as 语言管理 (LanguageContext)
    participant Wails as 前端 API (lib/api.ts)
    participant Service as 后端服务 (AppService)
    participant Manager as 模型管理器 (ModelManager)
    participant Mirror as 国内镜像 CDN (hf-mirror / ghfast)
    participant Disk as 本地文件系统 (DataDir/models)

    Note over User, i18n: 1. 语言切换交互
    User->>i18n: 点击侧边栏底部中英切换 (zh/en)
    i18n->>i18n: 更新状态并持久化 localStorage('vdub_locale')
    i18n-->>User: 全界面组件毫秒级重绘为目标语言

    Note over User, Disk: 2. 模型下载与断点续传
    User->>Wails: 点击下载模型 (如 large-v3-turbo, 镜像: hf-mirror)
    Wails->>Service: 调用 DownloadWhisperModel(name, mirror)
    Service->>Manager: 申请下载并加锁 (ActiveLock)
    Manager->>Disk: 检查已存在片段 (ggml-model.bin.tmp)
    Disk-->>Manager: 返回已有字节数 (如已下载 200MB)

    Manager->>Mirror: HTTP GET /resolve/main/ggml-model.bin (Header: Range: bytes=200000000-)
    Mirror-->>Manager: 206 Partial Content (Content-Range, Body Stream)

    loop 流式分块传输与进度推送 (Throttle 100ms)
        Manager->>Disk: 写入字节流块 (Append)
        Manager->>Service: 统计计算完成百分比、瞬时速率 (MB/s)
        Service->>Wails: 发送 Wails Event: whisper_download_progress
        Wails-->>User: 动态刷新进度环、速率及剩余时间
    end

    Mirror-->>Manager: 传输完成 (EOF)
    Manager->>Disk: 原子重命名 (Rename tmp -> bin)
    Manager->>Service: 释放任务锁，标记模型就绪
    Service->>Wails: 发送 Wails Event: whisper_download_done
    Wails-->>User: 弹出完成提示，按钮更新为「设为当前模型」
```

---

## 模型与运行时状态机图 (stateDiagram-v2)

```mermaid
stateDiagram-v2
    [*] --> NotInstalled: 首次运行/未检测到文件

    state NotInstalled {
        [*] --> Idle
        Idle --> Connecting: 发起下载请求
        Connecting --> Error: 域名解析/连接失败
        Error --> Idle: 重试
    }

    NotInstalled --> Downloading: 建立连接并收到 200/206

    state Downloading {
        [*] --> Transferring
        Transferring --> Transferring: 接收数据块并推算速率
        Transferring --> Paused: 用户手动暂停/网络异常
        Paused --> Transferring: 断点续传恢复
    }

    Downloading --> Verifying: 字节流读取完毕
    Downloading --> NotInstalled: 用户主动取消并清理文件

    state Verifying {
        [*] --> CheckHeader
        CheckHeader --> CheckSize: 检查 GGML 格式头标识
        CheckSize --> Valid: 尺寸吻合
        CheckSize --> Corrupt: 尺寸异常
        Corrupt --> [*]: 抛出异常回滚
    }

    Verifying --> Installed: 原子移动至正式目录
    Installed --> Active: 用户设为当前默认 ASR 引擎
    Active --> Installed: 切换至其他模型
    Installed --> NotInstalled: 用户确认删除模型
```

---

## 并发冲突与容灾保护场景

```mermaid
flowchart TD
    ReqA[线程 A: 请求下载 large-v3-turbo] --> LockCheck{互斥锁状态检查}
    ReqB[线程 B: 重复点击或批量请求下载同一模型] --> LockCheck

    LockCheck -->|线程 A 抢先持有锁| MarkActive[在 activeDownloads 注册并开启任务]
    LockCheck -->|线程 B 命中已存在键| RejectB[直接返回: 任务正在进行中, 附带当前进度]

    MarkActive --> DownloadWorker[后台 Goroutine 执行流式下载]
    DownloadWorker --> MonitorCancel{监听 Context Cancel 信号}

    UserCancel[用户点击取消按钮] --> TriggerCancel[调用 context.CancelFunc]
    TriggerCancel --> MonitorCancel

    MonitorCancel -->|收到取消信号| SafeClose[安全关闭响应流与文件句柄]
    SafeClose --> ReleaseLock[释放下载管理器互斥锁]
    SafeClose --> CleanOrKeep[保留 .tmp 以备续传或按需清理]
    ReleaseLock --> Done([结束])
```

---

## 国际化架构 (i18n) 设计

1. **存储设计**：
   - 键名：`localStorage.getItem('vdub_locale')`
   - 支持枚举：`'zh-CN'` (简体中文，默认) 与 `'en-US'` (English)
2. **上下文封装**：
   - 创建 `frontend/src/i18n/index.tsx` 提供 `LanguageProvider` 与 `useTranslation()`。
   - 提供 `t(key: string, params?: Record<string, string | number>): string` 函数，支持嵌套键路径及插值变量替换。
3. **语言包目录**：
   - `frontend/src/i18n/locales/zh-CN.ts`
   - `frontend/src/i18n/locales/en-US.ts`
4. **触发位置**：
   - 侧边栏 `Sidebar.tsx` 最底端，与暗黑/明亮主题切换按钮横向并列。
   - 采用精致的药丸微按钮，直观呈现当前语言与切换动作。

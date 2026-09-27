# vdub

vdub 是一款面向桌面端的视频字幕翻译与配音工具。它将语音转录、字幕翻译、语音合成和视频合成整合到一个工作流中，支持按任务查看处理进度与结果。

## 功能

- 从视频或音频中提取语音并生成字幕。
- 翻译字幕或文本，默认使用 Bing，也支持 Google 与 OpenAI 兼容服务。
- 使用 Edge TTS 或 OpenAI 兼容 TTS 生成配音。
- 将字幕封装为软字幕、烧录到画面，或与配音合成为视频。
- 管理翻译术语，并在任务面板查看进度、日志和处理结果。
- 配置语音识别、翻译和语音合成服务。

## 开发

项目由 Go 桌面端和 React、TypeScript 前端组成，使用 Wails 提供桌面应用能力。

启动开发环境：

```sh
make dev
```

构建应用：

```sh
make build
```

运行项目测试：

```sh
make test
```

更多构建目标可运行 `make help` 查看。语音识别和媒体处理依赖所选引擎及对应模型；翻译与语音合成服务可在应用设置中配置。

## 项目地址

[GitHub：ixugo/lark-studio](https://github.com/ixugo/lark-studio)

<p align="center">
  <img src="frontend/public/lark-logo.webp" alt="lark-studio logo" width="128" height="128">
</p>

<h1 align="center">lark-studio</h1>

<p align="center">A desktop workspace for subtitles, translation, and dubbing.</p>

<p align="center">English | <a href="README_ZH.md">简体中文</a></p>

lark-studio brings speech recognition, subtitle translation, speech synthesis, and video composition into one desktop application. Turn video or audio into subtitles, translate existing text, generate narration, or combine these steps into a complete dubbing workflow.

![demo](./docs/lark.gif)

## What you can do

- **Create subtitles from speech.** Transcribe video and audio with local Whisper.cpp models or an OpenAI-compatible transcription service.
- **Translate subtitles and text.** Use Bing, Google, or an OpenAI-compatible language model. Manage a glossary to keep recurring names and terms consistent in model-based translation.
- **Generate speech and dubbing.** Use Edge TTS or an OpenAI-compatible speech service, choose a voice and speaking rate, and enter custom model and voice names for your service. Preview compatible TTS voices directly in settings.
- **Produce subtitled videos.** Export subtitle files, embed selectable soft subtitles, or burn subtitles into the video. Combine translated speech with video for a dubbed result.
- **Use existing subtitle files.** Import SRT, VTT, or ASS files in the subtitle composition workspace, with single-language and bilingual layouts.
- **Process multiple files.** Select several files, choose a workflow preset, or save your own combination of transcription, translation, speech, and video output settings.
- **Follow each task.** View processing stages, progress, logs, and generated files from the task board.
- **Work in your preferred interface language.** Switch between English and Simplified Chinese, with light and dark themes.

## Supported engines

| Capability | Available options |
| --- | --- |
| Speech recognition | Local Whisper.cpp; OpenAI-compatible transcription API |
| Local speech models | Download and select models in the app, including the compact multilingual Whisper tiny model at approximately 75 MiB |
| Translation | Bing as the default recommendation; Google; OpenAI-compatible language models |
| Speech synthesis | Edge TTS; OpenAI-compatible speech API with custom model and voice names |
| Subtitle composition | SRT, VTT, and ASS input; single-language or bilingual subtitles |
| Video output | Subtitle files, soft subtitles, burned-in subtitles, and dubbed video |

## Typical workflows

| Starting point | Result |
| --- | --- |
| Video or audio | Transcribed and translated subtitles |
| Video | Translated speech and a composed video |
| Video | Dubbing in the original language without translation |
| Plain text or SRT text | Translation into a selected language |
| Text | Spoken audio |
| Video and existing subtitle files | A video with single-language or bilingual burned-in subtitles |

## Getting started

1. Open the engine settings and select your speech recognition, translation, and speech synthesis providers.
2. For local recognition, install the Whisper.cpp runtime and download a model from the speech recognition page. For a compatible API, enter the endpoint, model, and credentials required by your service.
3. Add your files or text in the workspace, select a preset, and adjust the language, voice, and output options.
4. Start the task and open the task board to follow its progress and access the results.

Local Whisper.cpp recognition runs on your computer after its runtime and model are installed. Online translation, speech synthesis, and transcription providers require network access and receive the content needed for their service. Available languages and voices depend on the selected engine.

## Development

The application uses Go and Wails for the desktop layer, with React and TypeScript for the interface. Run `make dev` to start the development environment.

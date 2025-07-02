# 🎬 Subcut

Subcut is a stack-based, scriptable CLI tool for video editing with full subtitle support and layout templating for platforms like YouTube, TikTok, and Instagram. Inspired by [Markut](https://github.com/tsoding/markut) and powered by [FFmpeg](https://ffmpeg.org/) and [faster-whisper](https://github.com/guillaumekln/faster-whisper).

> Edit and export platform-ready videos in seconds with a clean `.subcut` script.

---

## ✨ Features

- Stack-based scripting language for composable video edits
- Subtitle support (`.vtt`, `.srt`) with embedding or external export
- Native integration with `faster-whisper` for auto-transcription
- Platform templates (e.g. TikTok, YouTube) for resolution/layout
- Batch and modular editing with reusable blocks
- Burned-in captions with styling (font, size, background, etc.)

---

## 📦 Installation

Coming soon...

---

## 🚀 Quick Example

```text
template tiktok

push videos/intro.mp4
trim 00:00:00 00:00:10
caption subs/intro.vtt embed bottom
export out/clip1.mp4

push videos/main.mp4
caption subs/main.vtt vtt
export out/clip2.mp4

push out/clip1.mp4
push out/clip2.mp4
concat
export out/final.mp4
```

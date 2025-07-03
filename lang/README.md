# 📜 Subcut Script Language — Feature Specification

> This document defines all current and planned instructions supported by the `.subcut` stack-based scripting language for video editing.

---

## ✅ Core Stack Instructions (MVP)

### `push <video.mp4>`

Push a video file onto the stack.

---

### `trim <start> <end> <video.mp4>`

Trim the top videos on the stack to a time range.

Example:

```text
trim 00:00:05 00:00:30
```

**Note:** if <video.mp4> is provided, it will be used as the source for trimming.

### `export <file.mp4>`

Write the current top of the stack to the given path.

---

### `concat`

Merge a set of videos on the stack into one.

---

## 🔜 Phase 2 — Templates & Styling

### `template <name>`

Load layout/caption config for a specific platform.

Example:

```text
template tiktok
```

---

### `burn [position] [style options]`

Burn subtitles with visual styling.

Style options (examples):

- `font=Roboto-Bold`
- `size=36`
- `bg=true`
- `color=white`
- `outline=black`

Example:

```text
burn bottom font=Roboto size=32 bg=true
```

---

### `caption <file.vtt> [embed|vtt] [position]`

Attach subtitles to the current top video.

- `embed`: Burn into the video
- `vtt`: Export sidecar `.vtt` file
- `position`: `top`, `bottom`, or `x=Y y=Z`

Example:

```text
caption subs/v1.vtt embed bottom
```

---

## 🧩 Phase 3 — Modular Logic & Batch

### `block <name>` / `end`

Create an isolated scoped editing block.

```text
block intro
  push videos/intro.mp4
  trim 00:00:00 00:00:10
  caption subs/intro.vtt embed
  export out/intro.mp4
end
```

---

### `use_stack <name>`

(Planned) Restore a previously defined block or stack.

---

### `split_into_clips duration=<seconds> prefix=<name>`

Slice the current video into multiple short clips.

Example:

```text
split_into_clips duration=59 prefix=out/clip
```

---

### `batch folder=<dir> output=<dir> template=<name>`

Batch edit every video in a folder.

```text
batch folder=videos/ output=out/ template=tiktok
  trim 00:00:00 00:01:00
  caption auto embed
  export auto
end
```

---

## ✨ Optional / Advanced (Planned)

### `detect_speech`

Auto-cut video based on speech/silence detection.

---

### `normalize_audio`

Balance audio loudness across the video.

---

### `thumbnail_from frame=<number>`

Generate thumbnail image from a given frame.

---

### `music <bgm.mp3> [ducking]`

Add background music and enable ducking if needed.

---

### `style_caption`

Override caption font, size, and background.

---

## 🛠️ Utility Keywords (planned or debug)

- `#` for comments
- `print_stack` — for debugging stack content
- `log` — internal use for verbose mode

---

## 📌 Example Full Script

```text
template tiktok

push videos/intro.mp4
trim 00:00:00 00:00:10
caption subs/intro.vtt embed bottom

push videos/main.mp4
caption subs/main.vtt vtt

concat
burn bottom font=Roboto size=28 bg=true
export out/final_tiktok.mp4
```

---

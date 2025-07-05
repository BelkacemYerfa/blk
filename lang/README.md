# 📜 Subcut Script Language — Feature Specification

> This document defines all current and planned instructions supported by the `.subcut` stack-based scripting language for video editing.

---

## ✅ Core Stack Instructions (MVP)

### `push <video.mp4>`

Push a video file onto the stack.

Example:

```text
push "videos/intro.mp4"
```

---

### `trim <start> <end> <video.mp4>`

Trim the top videos on the stack to a time range.

Example:

```text
trim "00:00:05" "00:00:30"
```

**Note:** if <video.mp4> is provided, it will be used as the source for trimming.

### `export <file.mp4>`

Write the current top of the stack to the given path.

```text
export "out/video.mp4"
```

---

### `concat`

Merge a set of videos on the stack into one.

---

### `thumbnail_from [frame=<number> | time=<hh:mm:ss>] <file.png>`

Extract a thumbnail image at a given frame index.

Example:

```text
thumbnail_from 123 "out/cover.png"
```

```text
thumbnail_from "00:00:10" "out/cover.png"
```

---

### `set_track <name> { config }`

Define a named audio track to be reused across videos.

Options:

- `path=<file.mp3>` — Path to the audio file
- `volume=X` — Set music volume (e.g., `0.5`)
- `duck` — Auto-lower music during speech
- `loop` — Repeat track to match video length

Examples:

```text
set_track bg {
  path: "assets/beat.mp3"
  volume: 0.3
  duck: false
  loop: true
}
```

### `use_track <name> <video-path|last|first>`

Attach a previously defined track to a specific video or all videos.

Examples:

```text
use_track bg intro.mp4
use_track bg first
```

## 🔜 Phase 2 — Templates & Styling

### `template <name>`

Load layout/caption config for a specific platform.

Example:

```text
template tiktok
```

---

### `caption <file.vtt> [embed|vtt] [style options]`

Attach subtitles to the current top video.

- `embed`: Burn into the video
- `vtt`: Export sidecar `.vtt` file
- `position`: `top`, `bottom`, or `x=Y y=Z`

**Style Options:**

- `font=<font-name>` — Font family (e.g., `font=Roboto-Bold`, `font=Arial`)
- `size=<number>` — Font size in pixels (e.g., `size=36`, `size=24`)
- `bg=<true|false>` — Enable/disable background box behind text (e.g., `bg=true`)
- `color=<color>` — Text color (e.g., `color=white`, `color=#FF0000`, `color=red`)
- `outline=<color>` — Text outline/stroke color (e.g., `outline=black`, `outline=#000000`)

Example:

```text
caption subs/v1.vtt embed bottom
```

---

## 🧩 Phase 3 — Modular Logic & Batch

### `block <name>`

Create an isolated scoped editing block.

```text
block {
  push videos/intro.mp4
  trim 00:00:00 00:00:10
  caption subs/intro.vtt embed
  export out/intro.mp4
}
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

### `batch <input-dir> <output-dir> <template>`

Batch edit every video in a folder.

```text
batch "videos/" "out/" "tiktok" {
 trim 00:00:00 00:01:00
 caption auto embed
 export auto
}
```

---

## ✨ Optional / Advanced (Planned)

### `detect_speech`

Auto-cut video based on speech/silence detection.

---

### `normalize_audio`

Balance audio loudness across the video.

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

# 📜 Subcut Script Language — Feature Specification

> This document defines all current and planned instructions supported by the `.subcut` stack-based scripting language for video editing.

---

## ✅ Core Stack Instructions (MVP)

### `set <name> (<video-path> | <name> | { config })`

Define a named video to be used later.

Examples:

- for a video file or another named video:

```text
set intro "out/intro.mp4"
set outro intro
```

- for a configuration object:

```text
set bg_track {
  path: "assets/beat.mp3"
  volume: 0.3
  duck: false
  loop: true
}
```

---

### `trim <start> <end> (video-path?)`

Trim the top videos on the stack to a time range.

Example:

```text
trim "00:00:05" "00:00:30"
```

**Note:** if <video.mp4> is provided, it will be used as the source for trimming.

---

### `concat`

Merge a set of videos on the stack into one.

---

### `thumbnail_from [<frame-number> | <hh:mm:ss>] <file.png>`

Extract a thumbnail image at a given frame index.

Example:

```text
thumbnail_from 123 "out/cover.png"
```

```text
thumbnail_from "00:00:10" "out/cover.png"
```

---

### `use <name> on <video-path|last|first>`

Attach a previously defined track to a specific video or all videos.

Examples:

- for a specific video:

```text
use bg on "intro.mp4"
use bg on "first"
```

- for the last video on the stack:

```text
use bg
```

---

### `process <name> { <subcut-code> }`

Create an isolated scoped editing process.

```text
process {
  push "videos/intro.mp4"
  trim "00:00:00" "00:00:10"
  export "out/intro.mp4"
}
```

**Note:** You can't have another process inside a process.

---

### `for each <name> in <folder-path> [recurse] { ... }`

Iterate over every video file in a directory and apply a block of operations.

```text
for each file in "videos/" {
  push file
  trim 00:00:00 00:00:30
  export "out/{filename}_short.mp4"
}
```

#### Optional `recurse`

Add `recurse` after the path to recursively search in subdirectories.

```text
for each video in "projects/" recurse  {
  push video
  caption "subs/{filename}.vtt" embed
  export "out/processed/{filename}.mp4"
}
```

**Note:** You can add a level limit to recursion by specifying `recurse <level>`:

```text
for each video in "projects/" recurse 2 {
  push video
  caption "subs/{filename}.vtt" embed
  export "out/processed/{filename}.mp4"
}
```

#### Runtime value exposure

When using for each, the current file's path and metadata are made available within the loop's element scope through special read-only runtime properties:

| Properties | Description                                    |
| ---------- | ---------------------------------------------- |
| `filepath` | Full path to the current file                  |
| `filename` | File name without extension (e.g., `clip01`)   |
| `ext`      | File extension (e.g., `.mp4`, `.mov`)          |
| `meta`     | Metadata of the file (duration, resolution...) |

Example using metadata:

```text
for each f in "clips/" {
  if meta.duration < 5
    skip

  push f
  export "out/{filename}_final.mp4"
}
```

---

### `if <condition>`

Conditionally execute the next line or block.

```text
if meta.duration > 10
  trim 00:00:00 00:00:10
```

With block:

```text
if index == 0 {
  caption "subs/intro.vtt" embed
}
```

---

### `else if <condition>`

Chain additional conditions.

```text
if index == 0 {
  caption "subs/intro.vtt" embed
} else if index == 1 {
  caption "subs/second.vtt" embed
}
```

---

### `else`

Fallback if no previous conditions match.

```text
if index == 0 {
  caption "subs/intro.vtt" embed
} else {
  caption "subs/default.vtt" embed
}
```

---

### `skip`

Skips the current iteration (like `continue`).

```text
if meta.duration < 5
  skip
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

### `use_stack <name>`

(Planned) Restore a previously defined process or stack.

---

### `split_into_clips <seconds> <output-dir-path>`

Slice the current video into multiple short clips.

Example:

```text
split_into_clips 10 "out/clips/"
```

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

```

```

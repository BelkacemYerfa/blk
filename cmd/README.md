# 🖥️ Subcut CLI Specification

> The `subcut` CLI is the main interface for running Subcut scripts, generating subtitles, baking platform-ready videos, and batch-processing video files.

---

## 📌 Command: `subcut gen`

Generate subtitles using `faster-whisper`.

```bash
subcut gen <video.mp4> --output <subs.vtt> [--language <lang>]
```

### Options:

- `--output`: Destination `.vtt` file path
- `--language`: (Optional) language code (e.g. `en`, `fr`, `ar`)

---

## 🧩 Future Commands (Planned)

## 📌 Command: `subcut run`

Run a `.subcut` script.

```bash
subcut run path/to/script.subcut
```

### Description:

Parses and executes all stack-based commands inside the `.subcut` file in order.

---

## 📌 Command: `subcut bake`

Quickly generate a processed video using a template.

```bash
subcut bake <video.mp4> --subs <subs.vtt> --template <name>
```

### Description:

- Automatically applies:
  - Resolution/aspect ratio from template
  - Subtitle position/styling from template
- Shortcut for creating platform-ready exports

---

## 📌 Command: `subcut version`

Print the current version.

```bash
subcut version
```

### `subcut template list`

List all available templates.

```bash
subcut template list
```

---

### `subcut batch`

Run Subcut logic across a folder of videos.

```bash
subcut batch --folder <videos/> --template <tiktok>
```

---

### `subcut split`

Split a long video into clips.

```bash
subcut split <video.mp4> --duration 59 --output clips/
```

---

### `subcut preview`

Simulate and log script execution without writing outputs.

```bash
subcut preview path/to/script.subcut
```

---

## ✅ Current CLI Summary

| Command         | Description                                      |
| --------------- | ------------------------------------------------ |
| `gen`           | Generate subtitles via faster-whisper            |
| `run`           | (Planned) Execute a `.subcut` script             |
| `bake`          | (Planned) Bake video using template and subtitle |
| `version`       | (Planned) Show Subcut CLI version                |
| `template list` | (Planned) Show available templates               |
| `batch`         | (Planned) Process folder of videos               |
| `split`         | (Planned) Split video into short clips           |
| `preview`       | (Planned) Simulate script run without writing    |

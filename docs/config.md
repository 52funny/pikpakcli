## Configuration

The CLI reads the following fields from `config.yml`:

```yml
proxy:
username: xxx
password: xxx
open:
  download_dir:
  default: []
  text: []
  image: []
  video: []
  audio: []
  pdf: []
```

### Basic Fields

- `username`: your PikPak account username or phone number with country code such as `+861xxxxxxxxxx`.
- `password`: your PikPak account password.
- `proxy`: optional proxy URL such as `http://127.0.0.1:7890`.

> `proxy` must contain `://`.

### Open Settings

The `open` section is used by the interactive shell builtin `open`.

- `download_dir`: optional local cache directory for files that must be downloaded before opening.
- `default`: fallback local command used when no file-type-specific command is configured.
- `text`: local command used to open text and source files.
- `image`: local command used to open image files.
- `video`: local command used to open video files.
- `audio`: local command used to open audio files.
- `pdf`: local command used to open PDF files.

Each command field is a YAML string array. The first item is the executable name and the remaining items are its arguments.

If the command array contains `{path}`, it will be replaced with the local file path or remote media URL. If `{path}` is not present, the path or URL is appended to the end of the command automatically.

For video files, the shell `open` command prefers opening a remote media URL directly. Other file types are downloaded to the local cache directory before opening.

### Default Open Behavior

If the `open` section is not configured, the builtin `open` uses platform defaults:

- macOS: `text -> TextEdit`, `image/pdf -> Preview`, `video/audio -> IINA`, others -> `open`
- Linux: `xdg-open`
- Windows: `cmd /c start`

### Example

```yml
proxy: http://127.0.0.1:7890
username: +861xxxxxxxxxx
password: your-password
open:
  download_dir: ~/Downloads/pikpak-open
  default: ["open"]
  text: ["zed"]
  image: ["open", "-a", "Preview"]
  video: ["open", "-a", "IINA"]
  audio: ["open", "-a", "IINA"]
  pdf: ["open", "-a", "Preview"]
```

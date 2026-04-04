## 配置说明

CLI 会从 `config.yml` 中读取以下字段：

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

### 基础字段

- `username`：你的 PikPak 账号用户名，或者带区号的手机号，例如 `+861xxxxxxxxxx`。
- `password`：你的 PikPak 账号密码。
- `proxy`：可选代理地址，例如 `http://127.0.0.1:7890`。

> `proxy` 必须包含 `://`。

### Open 配置

`open` 配置段用于交互式 shell 中的内置 `open` 命令。

- `download_dir`：可选的本地缓存目录，用于存放打开前需要先下载的文件。
- `default`：当没有匹配到具体文件类型配置时使用的兜底本地命令。
- `text`：用于打开文本文件和源码文件的本地命令。
- `image`：用于打开图片文件的本地命令。
- `video`：用于打开视频文件的本地命令。
- `audio`：用于打开音频文件的本地命令。
- `pdf`：用于打开 PDF 文件的本地命令。

每个命令字段都使用 YAML 字符串数组。第一个元素是可执行程序名，后面的元素是它的参数。

如果命令数组中包含 `{path}`，运行时会将它替换为本地文件路径或远端媒体 URL。如果没有写 `{path}`，程序会自动把路径或 URL 追加到命令末尾。

对于视频文件，shell 中的 `open` 命令会优先直接打开远端媒体 URL。其他文件类型会先下载到本地缓存目录，再调用本地程序打开。

### 默认打开行为

如果没有配置 `open`，内置 `open` 会使用各平台默认行为：

- macOS：`text -> TextEdit`，`image/pdf -> Preview`，`video/audio -> IINA`，其他类型 -> `open`
- Linux：`xdg-open`
- Windows：`cmd /c start`

### 示例

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

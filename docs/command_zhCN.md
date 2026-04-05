# 命令使用方法

## 上传

- 将本地目录下的所有文件上传至 Movies 文件夹内

  ```bash
  pikpakcli upload -p Movies .
  ```

- 将本地目录下除了后缀名为`mp3`, `jpg`的文件上传至 Movies 文件夹内

  ```bash
  pikpakcli upload  -e .mp3,.jpg -p Movies .
  ```

- 指定上传的协程数目(默认为 16)

  ```bash
  pikpakcli -c 20 -p Movies .
  ```

- 使用 `-P` 标志设置 Pikpak 云上文件夹的 `id`

  ```bash
  pikpakcli upload -P AgmoDVmJPYbHn8ito1 .
  ```

## 下载

- 下载 `-p` 指向的目标。如果该目标是文件夹则递归下载，如果是文件则下载该文件

  ```bash
  pikpakcli download -p Movies
  pikpakcli download -p Movies/Peppa_Pig.mp4
  ```

- 把 `-p` 作为远端基路径，再拼接后面的参数。CLI 会自动判断目标是文件还是文件夹

  ```bash
  pikpakcli download -p Movies Peppa_Pig.mp4
  pikpakcli download -p Movies Cartoons
  pikpakcli download -p Movies Kids/Peppa_Pig.mp4
  ```

- 如果参数本身是绝对路径，则会覆盖 `-p`

  ```bash
  pikpakcli download -p Movies /TV/Peppa_Pig.mp4
  ```

- 限制同时下载的文件个数 (默认: 1)

  ```bash
  pikpakcli download -c 5 -p Movies
  ```

- 指定下载内容的输出目录

  ```bash
  pikpakcli download -p Movies -o Film
  ```

- 使用 `-g` 标志显示下载过程中的状态信息
  ```bash
  pikpakcli download -p Movies -o Film -g
  ```

## 分享

- 分享 Movies 下的所有文件的链接

  ```bash
  pikpakcli share -p Movies
  ```

- 分享指定文件的链接

  ```bash
  pikpakcli share Movies/Peppa_Pig.mp4
  ```

- 分享链接输出到指定文件

  ```bash
  pikpakcli share  --out sha.txt -p Movies
  ```

## 新建

### 新建文件夹

- 在 Movies 下新建文件夹 NewFolder

  ```bash
  pikpakcli new folder -p Movies NewFolder
  ```

### 新建 Sha 文件

- 在 Movies 下新建 Sha 文件

  ```bash
  pikpakcli new sha -p /Movies 'PikPak://美国队长.mkv|22809693754|75BFE33237A0C06C725587F87981C567E4E478C3'
  ```

### 新建磁力

- 新建磁力文件

  ```bash
  pikpakcli new url 'magnet:?xt=urn:btih:e9c98e3ed488611abc169a81d8a21487fd1d0732'
  ```

## 配额

- 获取 PikPak 云盘的空间

  ```bash
  pikpakcli quota -H
  ```

## 获取目录信息

- 获取根目录下面的所有文件信息

  ```bash
  pikpakcli ls -lH -p /
  ```

## 删除

- 按完整路径删除文件

  ```bash
  pikpakcli delete /Movies/Peppa_Pig.mp4
  ```

- 使用 `-p` 指定父目录后删除其中的文件或文件夹

  ```bash
  pikpakcli delete -p /Movies Peppa_Pig.mp4
  ```

- 在同一路径下同时删除多个文件或文件夹

  ```bash
  pikpakcli delete -p /Movies File1.mp4 File2.mp4
  ```

## 垃圾文件清理

- 使用默认垃圾文件规则递归扫描目录。如果用户配置目录中还没有规则文件，CLI 会自动从当前仓库下载默认规则。

  ```bash
  pikpakcli rubbish
  pikpakcli rubbish -p /Movies
  ```

- 默认只预览匹配结果，不会删除；加上 `-d` 后才会执行删除。

  ```bash
  pikpakcli rubbish -p /Movies
  pikpakcli rubbish -p /Movies -d
  ```

- 打开本地规则文件或规则目录。如果默认规则文件不存在，会先下载再打开。

  ```bash
  pikpakcli rubbish --open-rules
  pikpakcli rubbish --open-rules-dir
  ```

- 手动下载默认规则文件，或者指定自定义本地路径 / 远程 URL 作为规则来源。

  ```bash
  pikpakcli rubbish --download-rules
  pikpakcli rubbish --rules ~/.config/pikpakcli/rules/rubbish_rules.txt
  pikpakcli rubbish --rules https://raw.githubusercontent.com/52funny/pikpakcli/master/rules/rubbish_rules.txt
  ```

## 重命名

- 按完整路径重命名文件或文件夹

  ```bash
  pikpakcli rename /Movies/Peppa_Pig.mp4 Peppa_Pig_S01E01.mp4
  ```

- 重命名文件夹

  ```bash
  pikpakcli rename /Movies/Cartoons Kids
  ```

## 交互 Shell

- 启动交互式 shell

  ```bash
  pikpakcli shell
  ```

- 在 shell 中切换目录并查看当前目录文件

  ```bash
  pikpakcli shell
  cd "/Movies/Kids Cartoons"
  ls
  ```

- 在 shell 中打开远端文件到本地默认程序

  ```bash
  pikpakcli shell
  cd "/Movies"
  open Peppa_Pig.mp4
  ```

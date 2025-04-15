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

- 下载特定目录(如：`Movies` )下的所有文件

  ```bash
  pikpakcli download -p Movies
  ```

- 下载单个文件

  ```bash
  pikpakcli download -p Movies Peppa_Pig.mp4
  # or
  pikpakcli download Movies/Peppa_Pig.mp4
  ```

- 限制同时下载的文件个数 (默认: 3)

  ```bash
  pikpakcli download -c 5 -p Movies
  ```

- 指定下载文件的输出目录

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

- 删除指定文件

  ```bash
  pikpakcli delete Movies/Peppa_Pig.mp4
  ```

- 删除指定目录中的文件

  ```bash
  pikpakcli delete -p 文件夹路径 文件名
  ```

- 同时删除多个文件

  ```bash
  pikpakcli delete -p 文件夹路径 文件1 文件2
  ```

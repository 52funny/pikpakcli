# pikpakcli

PikPak 的命令行工具

## 配置文件

首先将项目中的 `config_example.yml` 配置一下，输入自己的账号密码

如果账号是手机号，手机号要以区号开头。如 `+861xxxxxxxxxx`

然后将其重命名为 `config.yml`

## 安装方法

### 从源码编译

首先你得拥有 go 的环境

[go install guide](https://go.dev/doc/install)

克隆项目

```bash
git clone https://github.com/52funny/pikpakcli
```

生成可执行文件

```bash
go build
```

运行

```bash
./pikpakcli
```

### 从 Release 下载

从 Release 下载你所需要的版本，然后运行。

## 使用方法

### 上传

将本地目录下的所有文件上传至 `pikpak` 根目录 Movies

```bash
./pikpakcli upload -p Movies .
```

将本地目录下除了后缀名为`mp3`, `jpg`的文件上传至 `pikpak` 根目录 Movies

```bash
./pikpakcli upload  -e .mp3,.jpg -p Movies .
```

指定上传的协程数目(默认为 16)

```bash
./pikpakcli -c 20 -p Movies .
```

使用 `-P` 指令来设置文件夹的 `id`

```bash
./pikpakcli upload -P AgmoDVmJPYbHn8ito1 .
```

### 下载

可以下载指定目录(如：`Movies` )下的所有文件

```bash
./pikpakcli download -p Movies
```

下载单个文件

```bash
./pikpakcli download -p Movies Peppa_Pig.mp4
```

OR

```bash
./pikpakcli download Movies/Peppa_Pig.mp4
```

可以限制下载的一次下载文件的个数 (默认: 3)

```bash
./pikpakcli download -c 5 -p Movies
```

可以指定文件夹的输出目录

```bash
./pikpakcli download -p Movies -o Film
```

### 分享

分享 `Movies` 下的所有文件的链接

```bash
./pikpakcli share -p Movies
```

分享指定文件的链接

```bash
./pikpakcli share Movies/Peppa_Pig.mp4
```

分享链接输出到指定文件

```bash
./pikpakcli share  --out sha.txt -p Movies
```

### 新建

#### 新建文件夹

在 /Movies 下新建文件夹 NewFolder

```bash
./pikpakcli new folder -p Movies NewFolder
```

#### 新建 Sha 文件

在 Movies 下新建 Sha 文件

```bash
./pikpakcli new sha -p /Movies 'PikPak://美国队长.mkv|22809693754|75BFE33237A0C06C725587F87981C567E4E478C3'
```

#### 新建磁力

### 获取配额

获取 pikpak 云盘的空间

```bash
./pikpakcli quota -H
```

### 获取目录信息

获取根目录下面的所有文件信息

```bash
./pikpakcli ls -lH -p /
```

新建磁力文件

```bash
./pikpakcli new url 'magnet:?xt=urn:btih:e9c98e3ed488611abc169a81d8a21487fd1d0732'
```

## 额外配置

### 使用 Proxy

打开 `config.yml` 配置 `proxy` 设置项目

```yml
proxy: http://host:port
```

> proxy 必须包含 `://`

### 内嵌 `config.yml` 到可执行文件

```bash
./pikpakcli embed -u config.yml
```

> -u 表示更新配置文件如果已经内嵌了 `config.yml`

### 查看命令提示

```bash
./pikpakcli help
```

### 查看指定命令的提示

```bash
./pikpakcli help upload
```

# PikPak CLI

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/52funny/pikpakcli)
![GitHub](https://img.shields.io/github/license/52funny/pikpakcli)

PikPakCli 是 PikPak 的命令行工具。

![Build from source code.](./images/build.gif)

## 安装方法

### 从源码编译

要从源代码构建该工具，请确保您的系统已安装 [Go](https://go.dev/doc/install) 环境。

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

## 配置文件

运行 `setup` 创建默认配置文件，并按提示输入账号密码：

```bash
pikpakcli setup
```

如果账号是手机号，手机号要以区号开头。如 `+861xxxxxxxxxx`

如果配置文件已经存在，`setup` 默认不会覆盖；需要重写时请加 `--force`。

配置文件将会优先从当前目录进行读取 `config.yml`，如果当前目录下不存在 `config.yml` 将会从用户的配置数据的默认根目录进行读取，各个平台的默认根目录如下：

- Linux: `$HOME/.config/pikpakcli`
- Darwin: `$HOME/Library/Application Support/pikpakcli`
- Windows: `%AppData%/pikpakcli`

可选的 `open` 配置段可以覆盖交互式 shell 中 `open` 内置命令针对不同文件类型使用的本地程序。

## 开始

之后你就可以运行 `ls` 指令来查看存储在 **PikPak** 上的文件了

```bash
./pikpakcli ls
```

## 用法

参阅 [Command](docs/command_zhCN.md) 查看更多的指令

## 贡献者

<a href = "https://github.com/52funny/pikpakcli/graphs/contributors">
  <img src = "https://contrib.rocks/image?repo=52funny/pikpakcli"/>
</a>

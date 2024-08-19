# PikPak CLI

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/52funny/pikpakcli)
![GitHub](https://img.shields.io/github/license/52funny/pikpakcli)

PikPakCli 是 PikPak 的命令行工具。

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

## 配置文件

首先将项目中的 `config_example.yml` 配置一下，输入自己的账号密码

如果账号是手机号，手机号要以区号开头。如 `+861xxxxxxxxxx`

然后将其重命名为 `config.yml`

配置文件将会优先从当前目录进行读取 `config.yml`，如果当前目录下不存在 `config.yml` 将会从用户的配置数据的默认根目录进行读取，各个平台的默认根目录如下：

- Linux: `$HOME/.config/pikpakcli`
- Darwin: `$HOME/Library/Application Support/pikpakcli`
- Windows: `%AppData%/pikpakcli`

## 开始

之后你就可以运行 `ls` 指令来查看存储在 **PikPak** 上的文件了

```bash
./pikpakcli ls
```

## 贡献者

<a href = "https://github.com/52funny/pikpakcli/graphs/contributors">
  <img src = "https://contrib.rocks/image?repo=52funny/pikpakcli"/>
</a>

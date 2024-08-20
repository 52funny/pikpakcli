# PikPak CLI

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/52funny/pikpakcli)
![GitHub](https://img.shields.io/github/license/52funny/pikpakcli)

English | [简体中文](https://github.com/52funny/pikpakcli/blob/master/README_zhCN.md)

PikPakCli is a command line tool for Pikpak Cloud.

## Installation

### Compiling from source code

First, make sure you have the Go environment set up.

[Go Install Guide](https://go.dev/doc/install)

Clone the project:

```bash
git clone https://github.com/52funny/pikpakcli
```

Build the project:

```bash
go build
```

Run:

```
./pikpakcli
```

### Download from Release

Download the executable file you need from the Releases page, then run it.

## Configuration

First, configure the `config_example.yml` file in the project, entering your account details.

If your account uses a phone number, it must be preceded by the country code, like `+861xxxxxxxxxx`.

Then, rename it to `config.yml`.

The configuration file will first be read from the current directory (`config.yml`). If it doesn't exist there, it will be read from the user's default configuration directory. The default root directories for each platform are:

- Linux: `$HOME/.config/pikpakcli`
- Darwin: `$HOME/Library/Application Support/pikpakcli`
- Windows: `%AppData%/pikpakcli`

## Get started

After that you can run the `ls` command to see the files stored on **PikPak**.

```bash
./pikpakcli ls
```

## Usage

See [Command](docs/command.md) for more commands information.

## Contributors

<a href = "https://github.com/52funny/pikpakcli/graphs/contributors">
  <img src = "https://contrib.rocks/image?repo=52funny/pikpakcli"/>
</a>

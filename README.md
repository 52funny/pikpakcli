# PikPak CLI

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/52funny/pikpakcli)
![GitHub](https://img.shields.io/github/license/52funny/pikpakcli)

English | [简体中文](https://github.com/52funny/pikpakcli/blob/master/README_zhCN.md)

PikPakCli is a command line tool for Pikpak Cloud.

![Build from source code.](./images/build.gif)

## Installation

### Compiling from source code

To build the tool from the source code, ensure you have [Go](https://go.dev/doc/install) installed on your system.

Clone the project:

```bash
git clone https://github.com/52funny/pikpakcli
```

Build the project:

```bash
go build
```

Run the tool:

```
./pikpakcli
```

### Build with Docker
You can also run `pikpakcli` using Docker.
Pull the Docker image:

```bash
docker pull 52funny/pikpakcli:master
```

Run the tool:
```bash
docker run --rm 52funny/pikpakcli:master --help
```

### Download from Release

Download the executable file you need from the [Releases](https://github.com/52funny/pikpakcli/releases) page, then run it.

## Configuration

First, configure the `config_example.yml` file in the project, entering your account details.

If your account uses a phone number, it must be preceded by the country code, like `+861xxxxxxxxxx`.

Then, rename it to `config.yml`.

The configuration file will first be read from the current directory (`config.yml`). If it doesn't exist there, it will be read from the user's default configuration directory. The default root directories for each platform are:

- Linux: `$HOME/.config/pikpakcli`
- Darwin: `$HOME/Library/Application Support/pikpakcli`
- Windows: `%AppData%/pikpakcli`

> **For Docker Users:** You need to mount the configuration file into the Docker container. For example, if your `config.yml` is located at `/path/to/your/config.yml`, you can run the Docker container like this:

```bash
docker run -v /path/to/your/config.yml:/root/.config/pikpakcli/config.yml pikpakcli:latest ls
# if your config.yml is in the project directory, you can just run:
docker run -v $PWD/config.yml:/root/.config/pikpakcli/config.yml pikpakcli:latest ls
```

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

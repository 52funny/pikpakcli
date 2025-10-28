# Docker Command Usage

For docker users, the most different part is linking the configuration file (i.e., `config.yml`) and folder you want to operate (e.g., `download` or `upload`) into the container.

## Upload

- Uploads all files in the local directory (e.g., `/path/to/upload`) to the `Movies` folder.

  ```bash
  # original cli: pikpakcli upload -p Movies .
  # Docker cli
  docker run --rm -v /path/to/config.yml:/root/.config/pikpakcli/config.yml -v /path/to/upload/:/upload pikpakcli:latest upload -p Movies /upload
  ```

- Upload files in local directory except for `mp3`, `jpg` to Movies folder.

  ```bash
  # original cli: pikpakcli upload -e .mp3,.jpg -p Movies .
  # Docker cli
  docker run --rm -v /path/to/config.yml:/root/.config/pikpakcli/config.yml -v /path/to/upload/:/upload pikpakcli:latest upload -e .mp3,.jpg -p Movies /upload
  ```

## Download

- Download all files in a specific directory (e.g. `Movies`).

```bash
  # original cli: pikpakcli download -p Movies
  # Docker cli
  # the option -o is used to specify the folder in container to save downloaded files
  docker run --rm -v /path/to/config.yml:/root/.config/pikpakcli/config.yml -v /path/to/download/:/download pikpakcli:latest download -p Movies -o /download
  ```

- Downloading a single file (e.g., `Movies/Peppa_Pig.mp4`).

```bash
  # original cli: pikpakcli download -p Movies Peppa_Pig.mp4
  # Docker cli
  docker run --rm -v /path/to/config.yml:/root/.config/pikpakcli/config.yml -v /path/to/download/:/download pikpakcli:latest download -p Movies Peppa_Pig.mp4 -o /download 
  ```


> Other download commands are omitted here, please refer to the original cli commands in [Command Usage](docs/command.md).


## Wrapper Script

We provide a wrapper script `docker_cli.sh` to simplify the docker command usage. You can run the script directly after setting up the `config.yml` file in the current directory. The script will create two folders `pikpak_downloads` and `pikpak_uploads` in the current directory for download and upload operations respectively.

```bash
# Make the script executable
chmod +x docker_cli.sh
# Run the script for upload
./docker_cli.sh upload -p Movies ./pikpak_uploads
# Run the script for download
./docker_cli.sh download -p Movies -o ./pikpak_downloads
```
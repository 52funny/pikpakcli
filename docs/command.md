# Command Usage

> For docker users, please refer to the [Docker Command Usage](docs/command_docker.md).

## Upload

- Uploads all files in the local directory to the Movies folder.

  ```bash
  pikpakcli upload -p Movies .
  ```

- Upload files in local directory except for `mp3`, `jpg` to Movies folder.

  ```bash
  pikpakcli upload  -e .mp3,.jpg -p Movies .
  ```

- Select the number of concurrent tasks for the upload (default is 16).

  ```bash
  pikpakcli -c 20 -p Movies .
  ```

- Use the `-P` flag to set the `id` of the folder on the Pikpak cloud.

  ```bash
  pikpakcli upload -P AgmoDVmJPYbHn8ito1 .
  ```

- Running `pikpakcli upload` without any local path arguments shows the command help.

## Download

- Download the target pointed to by `-p`. If it is a directory, download it recursively; if it is a file, download that file.

  ```bash
  pikpakcli download -p Movies
  pikpakcli download -p Movies/Peppa_Pig.mp4
  ```

- Use `-p` as the base remote path, then append the following argument to it. The CLI will decide whether the target is a file or a directory.

  ```bash
  pikpakcli download -p Movies Peppa_Pig.mp4
  pikpakcli download -p Movies Cartoons
  pikpakcli download -p Movies Kids/Peppa_Pig.mp4
  ```

- Use an absolute remote path in the argument to override `-p`.

  ```bash
  pikpakcli download -p Movies /TV/Peppa_Pig.mp4
  ```

- Limit the number of files that can be downloaded at the same time (default: 1).

  ```bash
  pikpakcli download -c 5 -p Movies
  ```

- Specify the output directory of downloaded files.

  ```bash
  pikpakcli download -p Movies -o Film
  ```

- Use the `-g` flag to display status information during the download process.

  ```bash
  pikpakcli download -p Movies -o Film -g
  ```

## Share

- Share links to all files under Movies.

  ```bash
  pikpakcli share -p Movies
  ```

- Share the link to the specified file.

  ```bash
  pikpakcli share Movies/Peppa_Pig.mp4
  ```

- Share link output to a specified file.

  ```bash
  pikpakcli share  --out sha.txt -p Movies
  ```

## New

### New Folder

- Create a new folder NewFolder under Movies

  ```bash
  pikpakcli new folder -p Movies NewFolder
  ```

### New Sha File

- Create a new Sha file under Movies.

  ```bash
  pikpakcli new sha -p /Movies 'PikPak://美国队长.mkv|22809693754|75BFE33237A0C06C725587F87981C567E4E478C3'
  ```

### New Magnet File

- Create new magnet file.

  ```bash
  pikpakcli new url 'magnet:?xt=urn:btih:e9c98e3ed488611abc169a81d8a21487fd1d0732'
  ```

## Quota

- Get space on your PikPak cloud drive.

  ```bash
  pikpakcli quota -H
  ```

## Ls

- Get information about all files in the root directory.

  ```bash
  pikpakcli ls -lH -p /
  ```

## Delete

- Delete a file by full path from the PikPak cloud.

  ```bash
  pikpakcli delete /Movies/Peppa_Pig.mp4
  ```

- Delete entries from a specific directory using the `-p` flag.

  ```bash
  pikpakcli delete -p /Movies Peppa_Pig.mp4
  ```

- Delete multiple entries under the same path.

  ```bash
  pikpakcli delete -p /Movies File1.mp4 File2.mp4
  ```

## Rubbish

- Scan a directory recursively with the default rubbish rules. If the rule file does not exist in the user config directory, the CLI downloads it from this repository automatically.

  ```bash
  pikpakcli rubbish
  pikpakcli rubbish -p /Movies
  ```

- Preview matched rubbish files without deleting them, then delete them with `-d`.

  ```bash
  pikpakcli rubbish -p /Movies
  pikpakcli rubbish -p /Movies -d
  ```

- Open the local rules file or the local rules directory. If the default rule file is missing, it is downloaded first and then opened.

  ```bash
  pikpakcli rubbish --open-rules
  pikpakcli rubbish --open-rules-dir
  ```

- Download the default rules file explicitly, or use a custom local path or remote URL as the rules source.

  ```bash
  pikpakcli rubbish --download-rules
  pikpakcli rubbish --rules ~/.config/pikpakcli/rules/rubbish_rules.txt
  pikpakcli rubbish --rules https://raw.githubusercontent.com/52funny/pikpakcli/master/rules/rubbish_rules.txt
  ```

## Rename

- Rename a file or folder by full path.

  ```bash
  pikpakcli rename /Movies/Peppa_Pig.mp4 Peppa_Pig_S01E01.mp4
  ```

- Rename a folder.

  ```bash
  pikpakcli rename /Movies/Cartoons Kids
  ```

## Shell

- Start the interactive shell.

  ```bash
  pikpakcli shell
  ```

- Change directory and list files in the current path.

  ```bash
  pikpakcli shell
  cd "/Movies/Kids Cartoons"
  ls
  ```

- Open a remote file from the shell with a local application.

  ```bash
  pikpakcli shell
  cd "/Movies"
  open Peppa_Pig.mp4
  ```

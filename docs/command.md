# Command Usage

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

## Download

- Download all files in a specific directory (e.g. `Movies`).

  ```bash
  pikpakcli download -p Movies
  ```

- Downloading a single file.

  ```bash
  pikpakcli download -p Movies Peppa_Pig.mp4
  # or
  pikpakcli download Movies/Peppa_Pig.mp4
  ```

- Limit the number of files that can be downloaded at the same time (default: 1)

  ```bash
  pikpakcli download -c 5 -p Movies
  ```

- Specify the output directory of the downloaded file.

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

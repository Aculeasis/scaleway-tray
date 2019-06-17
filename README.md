# Scaleway Tray
[![Build Status](https://travis-ci.org/Aculeasis/scaleway-tray.svg?branch=master)](https://travis-ci.org/Aculeasis/scaleway-tray)

Shows Scaleway instances info in the systray, using Scaleway API.

## Settings

See [SETTINGS.md](SETTINGS.md)

## Building

- **Windows**:
  - Install MinGW{-w64} and add bin path in to Path env.
  - `go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo`
- **Linux**:
  - `sudo apt-get install libgtk-3-dev libappindicator3-dev`

```bash
go get ./src
./build.{sh|bat}
```

Also, for running on Linux set `sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"`, see [this note](https://github.com/sparrc/go-ping#note-on-linux-support) for more details.

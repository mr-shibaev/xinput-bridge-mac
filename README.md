# xinput-bridge-mac

ASUS ROG Raikiri II over its 2.4 GHz dongle in CrossOver games on macOS.

The dongle shows up as a USB device with an XInput interface (the Xbox 360 wire protocol), which
macOS does not understand. `raikirid` reads it through libusb and writes the gamepad state to
`/tmp/raikiri.xinput`. The `xinput1_4.dll` from `dll/` is installed into a CrossOver bottle and
serves that state to games instead of the XInput built into Wine.

Games that use XInput work, directly or through their own `SDL2.dll`. DirectInput-only games and
Steam Input will not see the controller. Rumble is not forwarded to the controller yet.

## Requirements

- Go 1.27 or newer
- libusb, pkgconf and mingw-w64 from Homebrew

```
brew install go libusb pkgconf mingw-w64
```

## Run manually

Switch the controller to 2.4 GHz mode and plug the dongle in.

```
go run . -v
```

Prints buttons and sticks on every report. Without `-v` it runs silently. When the dongle is
missing the daemon waits for it and logs only state changes (connected, lost).

## Run as a daemon

```
make daemon
```

Builds the binary into `~/.local/bin/raikirid`, installs the launchd agent `com.shibaev.raikirid`
with KeepAlive and starts it. Log: `~/Library/Logs/raikirid.log`. Remove with
`make daemon-uninstall`.

## Install into a bottle

```
make install BOTTLE=Steam
```

Builds the DLLs (x64 and x86), copies them into the bottle's `system32` and `syswow64` and sets
the `native` override for `xinput1_4`, `xinput1_3` and `xinput9_1_0`. The daemon must be running
while the game runs. If CrossOver is not in `/Applications`, pass the path:
`make install CX_WINE=/path/to/CrossOver.app/Contents/SharedSupport/CrossOver/bin/wine`.

## Check inside the bottle

```
wine --bottle Steam control joy.cpl
```

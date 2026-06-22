# opendeck-bridge

WebSocket↔MIDI bridge with embedded OpenDeck configurator UI.

Runs on the Raspberry Pi CM5 alongside the pedalboard hardware, serving the web configurator and bridging SysEx messages between the browser and the MIDI device.

## Usage

```sh
opendeck-bridge -port "pedalboard" -addr ":8080"
```

Then open `http://cm5-dev.home:8080` in your browser.

## Building

```sh
# Native
make build

# Cross-compile for CM5 (arm64)
make build-arm64
```

## Architecture

```
Browser ←WebSocket→ opendeck-bridge ←ALSA MIDI→ pedalboard (hw:2,0,0)
         (port 8080)                  (rtmidi)
```

The OpenDeckUI static files are embedded in the binary via `go:embed`.

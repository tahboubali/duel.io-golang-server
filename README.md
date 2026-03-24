# duel.io Go Server

This repository contains the multiplayer server for `duel.io` and the integrated web hosting layer for the browser-playable client.

It is responsible for:

- WebSocket multiplayer traffic
- player registration and matchmaking
- duel lifecycle and state relaying
- serving the browser client from the same process

The Java game client itself lives in a separate repository. This repository hosts the built browser assets and exposes them on the same origin as the game server so the web client can connect directly to `/connect`.

## Requirements

- Go

## Run

Start the server:

```bash
go run .
```

The server listens on:

```text
http://127.0.0.1:8080/
```

Available endpoints:

- `/`: main browser game page
- `/panel.html`: compatibility alias that serves the same page
- `/connect`: WebSocket endpoint used by the desktop and browser clients

## What Lives Here

- [main.go](main.go): multiplayer server, matchmaking, duel coordination, and message handling
- [hosting.go](hosting.go): HTTP routing, static file hosting, WebSocket upgrade entrypoint, and request logging
- [web/index.html](web/index.html): browser host page for the web client
- [web/app/duel.io.jar](web/app/duel.io.jar): built Java client bundle served to the browser runtime
- [web/app/lib/](web/app/lib/): dependency jars required by the browser-served client
- [main_test.go](main_test.go): server test coverage

## Architecture Overview

### Multiplayer flow

1. A client connects to `/connect`.
2. The server upgrades that request to WebSocket.
3. The client registers with `new-player`.
4. The client enters matchmaking with `enter-duel`.
5. The server pairs compatible players and relays live duel messages such as:
   - `game-state`
   - `health-update`
   - `game-end`

### Web hosting flow

1. The browser opens `/`.
2. The server returns the hosted client page from `web/index.html`.
3. That page loads the built Java client bundle from `web/app/`.
4. The browser client connects back to `/connect` on the same origin.

This same-origin setup is why the old standalone proxy process is no longer needed.

## Project Layout

- `web/`: hosted browser assets
- `web/app/`: bundled Java client and dependency jars
- `web/index.html`: canonical browser entry page
- `main.go`: game server
- `hosting.go`: hosting integration layer

## Testing

This repository currently includes an integration-style test in [main_test.go](main_test.go) that expects the server to already be running on `localhost:8080`.

Start the server first:

```bash
go run .
```

Then, in another terminal, run:

```bash
go test -run TestServer_Run -count=1
```

For a quick manual check, run the server and open [http://127.0.0.1:8080/](http://127.0.0.1:8080/).

## Related Repository

The Java client source lives separately in:

- [duel.io Java client repository](https://github.com/tahboubali/duel.io)

That repository owns the Java source and Maven build. This repository owns the live server and the hosted browser delivery of the built client.

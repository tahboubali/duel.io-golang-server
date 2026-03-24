# duel.io Web Embed

This folder contains a browser-hosted version of the Java 8 client.

## Build the web bundle

```bash
./scripts/build-web-bundle.sh
```

That produces:

- `web/app/duel.io.jar`
- dependency jars under `web/app/lib/`

## Run locally

```bash
go run .
```

Then open:

- `http://127.0.0.1:8080/` for the main game panel
- `http://127.0.0.1:8080/panel.html` also works as a compatibility alias

## Notes

- The panel uses CheerpJ to run the Java client directly in-browser.
- The Go server now serves the browser assets and the `/connect` WebSocket endpoint from the same process.
- `/` is the canonical page you can embed in another site with an iframe.

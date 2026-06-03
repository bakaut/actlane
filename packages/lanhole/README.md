# Lanhole

`lanhole` is a separate file-transfer CLI for moving an already-created archive, for example `actlane-pack.zip`.

It is intentionally outside the `actlane` CLI boundary:

- `actlane` creates, validates, generates, plans, applies, removes, and serves pack/runtime artifacts.
- `lanhole` sends and receives one file over LAN or through an already-running relay.

`lanhole` does not include broker hosting, ACME, Docker, Docker Compose, deploy manifests, metrics, or permanent inbox/storage.

## Build

```bash
go test ./...
go build -o dist/lanhole ./cmd/lanhole
```

## LAN Transfer

Sender:

```bash
lanhole send ./actlane-pack.zip
```

Receiver:

```bash
lanhole recv 123-orange-river-candle --out ./downloads
```

The sender prints the one-time code. The receiver uses that code. The file metadata and content are encrypted end to end after PAKE handshake.

Useful flags:

```bash
lanhole send --code 123-orange-river ./actlane-pack.zip
lanhole recv --yes --out ./downloads 123-orange-river
lanhole recv --overwrite 123-orange-river
```

## Existing Relay

Use this only when you already have a compatible relay broker available:

```bash
lanhole send --transport tls --broker relay.example.com:443 ./actlane-pack.zip
lanhole recv --out ./downloads 'lanhole://relay.example.com:443/SESSION?transport=tls#CODE'
```

Supported relay transports in this MVP: `tcp`, `tls`.

## After Receive

Use the received archive with Actlane:

```bash
actlane pack inspect ./downloads/actlane-pack.zip
actlane generate ./downloads/actlane-pack.zip --target codex
```

## Security Notes

- LAN peers are not trusted; the human code is used for PAKE, not as a direct encryption key.
- The receiver denies overwrite by default and never auto-opens files.
- Relay mode is synchronous and client-side only in this package.
- Before using an internet relay in a company environment, confirm the relay policy with your security team.

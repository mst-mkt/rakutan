# rakutan

Answer ToyoNet-ACE quizzes comfortably, right from your terminal.

> [!NOTE]
> This is an unofficial tool. Changes on the ToyoNet-ACE side may break it or cause unintended actions. Use it at your own risk.

> [!WARNING]
> The AI suggestion feature is intended for courses and surveys where AI use is permitted. Using it in courses that prohibit AI may affect your grades or credits. It is no different from pasting a question into an AI chat yourself. Check each course's policy and take responsibility for how you use it.

## Usage

```nushell
nix run github:mst-mkt/rakutan
```

```nushell
go run github.com/mst-mkt/rakutan@latest
```

## Requirements

- Credentials and the session are stored in the system keyring (Secret Service on Linux, Keychain on macOS). Without a keyring, you will be asked to log in each time the session expires.
- AI suggestions require Claude Code.

## License

[MIT](./LICENSE)

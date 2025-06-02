[![Go Report][0]][1]
[![Go Version][2]][3]
[![License][4]][5]
[![Docker][6]][7]
[![Test][8]][9]

# text-to-speech bot

A simple text-to-speech bot for Discord written in Go. This bot listens to messages in a channel.

CLI Flags:
- `--config-path=your-config-path`: Path to the config file.
- `--sync-commands=true`: Synchronize commands with the discord.

This bot is under active development and is not yet feature complete.
It currently supports the following engines:
- [Google Cloud Text-to-Speech API][10].

## Usage

1. Create a Discord application at the [Discord Developer Portal][11].
2. Create a bot user for the application.
3. Copy the bot token and add it to the configuration file.
4. Create a service account in the Google Cloud Console and download the JSON key file.
5. Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of the JSON key file.
6. Install the bot by inviting it to your server using the OAuth2 URL generated in the Discord Developer Portal.
7. fill the `config.toml` file with your bot token and any other configuration options you want to set. The `dev_guilds` field can be left empty to sync commands globally or filled with specific guild IDs to sync commands only to those guilds.

    Example `config.toml`:
    ```toml
    [log]
    level = "info"
    format = "text"
    add_source = true

    [bot]
    dev_guilds = []
    token = "YOUR_BOT_TOKEN_HERE"
    ```

8. Run the bot with the command:
   ```bash
   go run main.go --config-path=path/to/config.toml --sync-commands=true
   ```
9. Execute `/join` in a text channel to make the bot join the voice channel you are in.


## Configuration

See the [config.toml.example][12] file for an example configuration file. The configuration file is in TOML format and can be customized to your needs.

## License

The bot template is licensed under the [Apache License 2.0][5].


[0]: https://goreportcard.com/badge/github.com/makeitchaccha/text-to-speech
[1]: https://goreportcard.com/report/github.com/makeitchaccha/text-to-speech

[2]: https://img.shields.io/github/go-mod/go-version/makeitchaccha/text-to-speech
[3]: https://golang.org/doc/devel/release.html

[4]: https://img.shields.io/github/license/makeitchaccha/text-to-speech
[5]: LICENSE

[6]: https://github.com/makeitchaccha/text-to-speech/actions/workflows/docker.yml/badge.svg
[7]: https://github.com/makeitchaccha/text-to-speech/actions/workflows/docker.yml

[8]: https://github.com/makeitchaccha/text-to-speech/actions/workflows/test.yml/badge.svg
[9]: https://github.com/makeitchaccha/text-to-speech/actions/workflows/test.yml

[10]: https://cloud.google.com/text-to-speech

[11]: https://discord.com/developers/applications

[12]: /config.example.toml
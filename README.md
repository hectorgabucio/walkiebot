# Walkiebot

A Discord bot that listens for voice messages in text channels and streams them via WebRTC to web browsers in real-time.

## Overview

Walkiebot acts as a bridge between Discord voice messages and WebRTC, allowing you to:
- Listen for voice messages in Discord text channels
- Download and parse voice message audio files (OGG/Opus format)
- Stream audio in real-time to web browsers via WebRTC

## Architecture

```
Discord Text Channel
    ↓
User sends voice message
    ↓
Bot detects voice message attachment
    ↓
Bot downloads OGG/Opus file
    ↓
Bot extracts Opus packets from OGG container
    ↓
Bot streams Opus packets via WebRTC → Browser/Client
```

## Features

- **Voice Message Detection**: Automatically detects voice messages in the first text channel
- **OGG/Opus Parsing**: Extracts Opus audio packets from Discord's OGG container format
- **WebRTC Streaming**: Streams audio to web clients via WebRTC peer connections
- **Continuous Streaming**: Maintains continuous RTP sequence numbers across multiple voice messages
- **Real-time Playback**: Streams voice messages to your browser as they're sent

## Prerequisites

- Go 1.21 or later
- A Discord bot token
- Discord bot with message reading permissions

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd walkiebot
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file in the root directory:
```env
DISCORD_TOKEN=your_discord_bot_token_here
```

4. Get your Discord bot token:
   - Go to https://discord.com/developers/applications
   - Create a new application or select an existing one
   - Go to the "Bot" section
   - Create a bot and copy the token
   - Enable the "Message Content Intent" in the Bot settings

5. Invite your bot to your Discord server with the following permissions:
   - Read Messages
   - View Channels
   - Send Messages (optional)

## Usage

1. Run the bot:
```bash
go run .
```

2. The bot will:
   - Connect to Discord
   - Find the first text channel in your server
   - Set up a WebRTC peer connection
   - Wait for voice messages

3. For WebRTC connection:
   - Open `webrtc-client.html` in your browser
   - Click "Generate Offer" to create a WebRTC offer
   - Copy the base64 offer using the "📋 Copy" button
   - Paste it into `offer.txt` file and save
   - The bot will automatically detect it and output an SDP answer
   - Copy the answer from the bot's output
   - Paste it into the web client's "Answer" field and click "Set Answer & Connect"

4. Send voice messages:
   - Send a voice message in the text channel the bot is monitoring
   - The bot will automatically detect it, download it, and stream it to your browser
   - Multiple voice messages will play in sequence with continuous audio streaming

5. The bot will continue running until you press `CTRL-C`.

## WebRTC Client

The included `webrtc-client.html` provides a complete WebRTC client interface:
- Generate WebRTC offers with one click
- Copy offers/answers to clipboard easily
- Automatic audio playback when connected
- Connection status indicators
- Works in any modern browser

## Dependencies

- [discordgo](https://github.com/bwmarrin/discordgo) - Discord API library
- [pion/webrtc](https://github.com/pion/webrtc) - WebRTC implementation
- [pion/rtp](https://github.com/pion/rtp) - RTP packet handling
- [godotenv](https://github.com/joho/godotenv) - Environment variable management

## Project Structure

```
walkiebot/
├── main.go                    # Main entry point, Discord bot and voice message handling
├── webrtc-client.html         # WebRTC client for browser
├── offer.txt                  # File for pasting WebRTC offers (auto-cleared after use)
├── internal/
│   ├── signal/
│   │   └── signal.go         # SDP offer/answer encoding/decoding, file I/O
│   └── webrtcserver/
│       └── webrtc.go         # WebRTC peer connection setup
├── go.mod                     # Go module dependencies
└── README.md                  # This file
```

## How It Works

1. **Discord Connection**: The bot connects to Discord using your bot token and identifies text channels.

2. **Voice Message Detection**: The bot listens for new messages in the first text channel and detects voice message attachments (`.ogg` or `.opus` files).

3. **Audio Download**: When a voice message is detected, the bot downloads the OGG/Opus audio file from Discord's CDN.

4. **OGG Parsing**: The bot parses the OGG container format to extract individual Opus audio packets.

5. **RTP Streaming**: Opus packets are wrapped in RTP format with continuous sequence numbers and timestamps, then streamed to the WebRTC track.

6. **WebRTC Playback**: The browser receives the RTP packets, decodes the Opus audio, and plays it through the audio element.

## Technical Details

- **Audio Format**: Discord voice messages are in OGG container format with Opus codec
- **Sample Rate**: 48kHz
- **RTP Payload Type**: 111 (Opus)
- **Sequence Numbers**: Continuous across multiple voice messages for seamless playback
- **Timestamps**: Increment by 960 samples per packet (20ms frames at 48kHz)

## Notes

- The bot uses file-based signaling (`offer.txt`) to avoid terminal paste issues with long base64 strings
- Voice messages are streamed in real-time as they're received
- Multiple voice messages play sequentially with continuous RTP sequence numbers
- The bot only processes voice messages from non-bot users
- Make sure your Discord bot has the "Message Content Intent" enabled

## Troubleshooting

- **No audio in browser**: Check browser console (F12) for errors, ensure audio isn't muted
- **Voice messages not detected**: Verify the bot has "Read Messages" permission in the channel
- **WebRTC connection fails**: Check that the offer/answer exchange completed successfully
- **Only first message plays**: This was fixed by maintaining continuous RTP sequence numbers

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

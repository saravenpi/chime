# Chime 🔔

A modern, terminal-based iMessage client for macOS built with Go and Bubble Tea.

![Chime Demo](https://img.shields.io/badge/platform-macOS-blue)
![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- 💬 **View and send iMessages** directly from your terminal
- 👥 **Contact management** with local YAML-based storage
- 🌐 **IRC Client** - Connect to IRC servers, join channels, and chat in real-time
- 🖥️ **IRC Server** - Host your own IRC server that others can connect to
- 🎨 **Beautiful TUI** powered by Bubble Tea and Lipgloss
- 🔍 **Search and filter** through conversations
- 📬 **Unread filter** - Toggle to view only unread messages with 'u' key
- ⚡ **Real-time contact name resolution** with live UI updates
- 🔄 **Auto-refresh** - Conversations update every 5 seconds
- 🌐 **Multiple contact sources**: local contacts, macOS Contacts app, and system AddressBook
- 📱 **Group chat support** with multiple sending strategies
- 🔐 **Read-only database access** for safety
- ⚡ **Quick contact add** - Add contacts directly from conversations
- 💌 **Start new conversations** - Message any phone number or email directly

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/saravenpi/chime/master/install.sh | bash
```

This will:
- Download and build Chime
- Install to `~/.local/bin/chime`
- Verify prerequisites (Go, macOS)

### Prerequisites

- macOS (required for iMessage integration)
- Go 1.25 or later
- Messages.app signed in to iMessage
- Full Disk Access permission for Terminal/iTerm

### Build from source

```bash
git clone https://github.com/saravenpi/chime.git
cd chime
go build -o chime
./chime
```

### Grant Full Disk Access

1. Open System Settings → Privacy & Security → Full Disk Access
2. Add your terminal app (Terminal.app or iTerm.app)
3. Restart your terminal

## Usage

```bash
# Start Chime TUI (iMessage client)
./chime

# Start IRC server mode
./chime server

# Show help
./chime help

# Show version
./chime version
```

### Navigation

**Main Menu:**
- `↑↓/jk` - Navigate between Conversations, Contacts, and IRC
- `Enter` - Select option
- `q` - Quit

**Conversations:**
- `↑↓/jk` - Navigate conversations
- `Enter` - Open conversation
- `n` - Start new conversation
- `/` - Search conversations
- `u` - Toggle unread filter
- `r` - Refresh
- `Esc` - Back to menu

**Messages:**
- `↑↓/jk` - Scroll messages
- `n` or `c` - Compose new message
- `a` - Add contact (for unknown numbers)
- `Ctrl+S` - Send message
- `r` - Refresh
- `Esc` - Back to conversations

**Contacts:**
- `↑↓/jk` - Navigate contacts
- `n` or `a` - Add new contact
- `Enter` - Edit contact
- `d` - Delete contact
- `/` - Search contacts
- `Esc` - Back to menu

**Contact Form:**
- `Tab/↑↓` - Navigate fields
- `Ctrl+S` - Save contact
- `Esc` - Cancel

**IRC Servers:**
- `↑↓/jk` - Navigate servers
- `Enter` - View channels (if connected)
- `a` or `n` - Add new server
- `c` - Connect to server
- `x` - Disconnect from server
- `d` - Delete server
- `/` - Search servers
- `Esc` - Back to menu

**IRC Channels:**
- `↑↓/jk` - Navigate channels
- `Enter` - Open chat
- `j` - Join new channel
- `p` - Part from channel
- `r` - Refresh channel list
- `Esc` - Back to servers

**IRC Chat:**
- `↑↓/jk` - Scroll messages
- `Enter` - Focus input field
- `Ctrl+S` - Send message
- `r` - Refresh messages
- `Esc` - Back to channels (or unfocus input)

## Contact Storage

Contacts are stored locally in `~/.chime/contacts/` as YAML files. Each contact has:

- **Name** (required)
- **Phone Numbers** (up to 3)
- **Email Addresses** (up to 3)

Example contact file (`~/.chime/contacts/John Doe.yml`):

```yaml
name: John Doe
phone_numbers:
  - +1234567890
  - +0987654321
emails:
  - john@example.com
```

### Contact Resolution Priority

Chime uses a multi-tiered approach to resolve contact names:

1. **Local contacts** (`~/.chime/contacts/`) - Fastest
2. **macOS Contacts app** (via AppleScript) - Cached for performance
3. **System AddressBook database** - Fallback

## IRC Support

Chime now includes full IRC (Internet Relay Chat) support, allowing you to connect to IRC servers and participate in chat rooms alongside your iMessage conversations.

### IRC Server Configuration

IRC server configurations are stored in `~/.chime/irc/servers.json`. You can add servers through the UI:

1. Select **🌐 IRC** from the main menu
2. Press `a` or `n` to add a new server
3. Fill in the server details:
   - **Server Name**: A friendly name (e.g., "Libera Chat")
   - **Host**: Server address (e.g., `irc.libera.chat`)
   - **Port**: Usually 6667 (plain) or 6697 (SSL)
   - **SSL**: Enter `yes` for encrypted connections
   - **Nickname**: Your IRC nickname
   - **Password**: Server password (optional, leave empty if not needed)

### Using IRC

**Connect to a Server:**
1. Navigate to the IRC Servers list
2. Select a server
3. Press `c` to connect

**Join a Channel:**
1. Once connected, press `Enter` on the server
2. Press `j` to join a new channel
3. Enter the channel name (e.g., `#golang` or just `golang`)

**Chat in a Channel:**
1. Select a channel and press `Enter`
2. Press `Enter` again to focus the input field
3. Type your message and press `Ctrl+S` to send
4. Press `Esc` to unfocus the input or go back

**Popular IRC Networks:**
- **Libera Chat** (`irc.libera.chat:6697` SSL) - Open source projects
- **OFTC** (`irc.oftc.net:6697` SSL) - Community projects
- **EFnet** (`irc.efnet.org:6667`) - One of the original IRC networks

## IRC Server Mode

Chime can also run as a standalone IRC server that others can connect to using any standard IRC client.

### Starting the Server

```bash
./chime server
```

On first run, this creates a default configuration file at `~/.chime.yml` with these settings:

```yaml
server:
  port: 6667
  host: 0.0.0.0
  name: chime.local
  description: Chime IRC Server
  motd: Welcome to Chime IRC Server!
```

### Configuration Options

Edit `~/.chime.yml` to customize your server:

- **port**: The port to listen on (default: 6667, SSL typically uses 6697)
- **host**: Bind address (0.0.0.0 for all interfaces, 127.0.0.1 for localhost only)
- **name**: Your server's hostname (used in messages)
- **description**: Server description shown to clients
- **motd**: Message of the Day displayed when users connect

### Connecting to Your Server

Users can connect with any IRC client:

```bash
# Using irssi
irssi -c localhost -p 6667

# Using weechat
/connect localhost/6667

# Using the chime client itself
# Add server: localhost:6667 (no SSL for local testing)
```

### Server Features

- **Multi-user support**: Multiple clients can connect simultaneously
- **Channel management**: Create and join channels (e.g., #general, #random)
- **Private messages**: Send direct messages between users
- **Standard IRC commands**: NICK, JOIN, PART, PRIVMSG, QUIT, TOPIC, WHO, MODE, NAMES
- **Thread-safe**: Concurrent connection handling with proper synchronization

## Architecture

### Three-Layer Design

1. **Models Layer** (`internal/models/`)
   - Data structures for chats, messages, contacts, and IRC entities

2. **Data Layer** (`internal/imessage/`, `internal/contacts/`, `internal/irc/`, `internal/ircserver/`, `internal/config/`)
   - Read-only SQLite access to iMessage database
   - AppleScript integration for sending messages
   - YAML-based contact storage and retrieval
   - IRC client connection management and message handling
   - IRC server implementation with channel and user management
   - JSON-based IRC client server configuration storage
   - YAML-based server configuration (`~/.chime.yml`)

3. **UI Layer** (`internal/ui/`)
   - Bubble Tea components for interactive TUI
   - Menu, conversations, messages, contacts, IRC servers, channels, and chat views

### Key Technical Details

- **Database**: Read-only access to `~/Library/Messages/chat.db`
- **Message Sending**: AppleScript integration with Messages.app
- **Contact Caching**: Thread-safe in-memory cache with `sync.RWMutex`
- **Async Operations**: Contact lookups run in background goroutines
- **Live Updates**: UI refreshes as contact names are resolved

## How It Works

### Reading Messages

Chime reads from your iMessage SQLite database at `~/Library/Messages/chat.db`. All database access is read-only to ensure safety.

### Sending Messages

Messages are sent via AppleScript commands to Messages.app. For group chats, Chime tries multiple strategies:
1. Send by chat ID (most reliable)
2. Send by group chat name
3. Create new chat with participants

### Contact Name Resolution

When displaying a phone number or email:
1. Check local `~/.chime/contacts/` first (instant)
2. Query macOS Contacts app via AppleScript (cached)
3. Fall back to system AddressBook database

Names load asynchronously and update the UI live as they're found.

## Development

### Project Structure

```
chime/
├── main.go                    # Entry point & server mode
├── internal/
│   ├── contacts/              # Contact storage & retrieval
│   │   └── contacts.go
│   ├── imessage/              # iMessage integration
│   │   ├── database.go        # Read messages from SQLite
│   │   └── send.go            # Send via AppleScript
│   ├── irc/                   # IRC client integration
│   │   ├── manager.go         # Connection & message handling
│   │   └── storage.go         # Server configuration storage
│   ├── ircserver/             # IRC server implementation
│   │   └── server.go          # Server, channels, user management
│   ├── config/                # Configuration management
│   │   └── config.go          # YAML config for server mode
│   ├── models/                # Data models
│   │   └── types.go
│   └── ui/                    # Bubble Tea UI components
│       ├── menu.go            # Main menu
│       ├── conversations.go   # Conversation list
│       ├── messages.go        # Message thread
│       ├── contacts_list.go   # Contact list
│       ├── contact_form.go    # Add/edit contact form
│       ├── irc_servers.go     # IRC server list
│       ├── irc_channels.go    # IRC channel list
│       ├── irc_chat.go        # IRC chat view
│       └── styles.go          # Lipgloss styles
├── go.mod
└── README.md
```

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
- [girc](https://github.com/lrstanley/girc) - IRC client library
- [irc-go](https://github.com/ergochat/irc-go) - IRC protocol library for server
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## Limitations

- **macOS only** - iMessage database is platform-specific
- **Messages.app required** - Must be running and signed in for sending
- **Read-only messages** - Cannot edit or delete existing messages
- **No push notifications** - Manual refresh required for new messages
- **Group chat limitations** - Cannot create new group chats, only reply to existing ones

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Author

**saravenpi**

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- Inspired by terminal-based messaging clients
- Thanks to the Go and macOS communities

---

**Note**: This is an unofficial client and is not affiliated with Apple Inc.

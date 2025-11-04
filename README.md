# Chime 🔔

A modern, terminal-based iMessage client for macOS built with Go and Bubble Tea.

![Chime Demo](https://img.shields.io/badge/platform-macOS-blue)
![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-MIT-green)

## Features

- 💬 **View and send iMessages** directly from your terminal
- 👥 **Contact management** with local YAML-based storage
- 🎨 **Beautiful TUI** powered by Bubble Tea and Lipgloss
- 🔍 **Search and filter** through conversations
- 📬 **Unread filter** - Toggle to view only unread messages with 'u' key
- ⚡ **Real-time contact name resolution** with live UI updates
- 🔄 **Auto-refresh** - Conversations update every 5 seconds
- 🌐 **Multiple contact sources**: local contacts, macOS Contacts app, and system AddressBook
- 📱 **Group chat support** with multiple sending strategies
- 🔐 **Read-only database access** for safety
- ⚡ **Quick contact add** - Add contacts directly from conversations

## Installation

### Quick Install (Recommended)

\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/saravenpi/chime/master/install.sh | bash
\`\`\`

This will:
- Download and build Chime
- Install to \`~/.local/bin/chime\`
- Verify prerequisites (Go, macOS)

### Prerequisites

- macOS (required for iMessage integration)
- Go 1.25 or later
- Messages.app signed in to iMessage
- Full Disk Access permission for Terminal/iTerm

### Build from source

\`\`\`bash
git clone https://github.com/saravenpi/chime.git
cd chime
go build -o chime
./chime
\`\`\`

### Grant Full Disk Access

1. Open System Settings → Privacy & Security → Full Disk Access
2. Add your terminal app (Terminal.app or iTerm.app)
3. Restart your terminal

## Usage

\`\`\`bash
# Start Chime
./chime

# Show help
./chime help

# Show version
./chime version
\`\`\`

### Navigation

**Main Menu:**
- \`↑↓/jk\` - Navigate between Conversations and Contacts
- \`Enter\` - Select option
- \`q\` - Quit

**Conversations:**
- \`↑↓/jk\` - Navigate conversations
- \`Enter\` - Open conversation
- \`/\` - Search conversations
- \`u\` - Toggle unread filter
- \`r\` - Refresh
- \`Esc\` - Back to menu

**Messages:**
- \`↑↓/jk\` - Scroll messages
- \`n\` or \`c\` - Compose new message
- \`a\` - Add contact (for unknown numbers)
- \`Ctrl+S\` - Send message
- \`r\` - Refresh
- \`Esc\` - Back to conversations

**Contacts:**
- \`↑↓/jk\` - Navigate contacts
- \`n\` or \`a\` - Add new contact
- \`Enter\` - Edit contact
- \`d\` - Delete contact
- \`/\` - Search contacts
- \`Esc\` - Back to menu

**Contact Form:**
- \`Tab/↑↓\` - Navigate fields
- \`Ctrl+S\` - Save contact
- \`Esc\` - Cancel

## Contact Storage

Contacts are stored locally in \`~/.chime/contacts/\` as YAML files. Each contact has:

- **Name** (required)
- **Phone Numbers** (up to 3)
- **Email Addresses** (up to 3)

Example contact file (\`~/.chime/contacts/John Doe.yml\`):

\`\`\`yaml
name: John Doe
phone_numbers:
  - +1234567890
  - +0987654321
emails:
  - john@example.com
\`\`\`

### Contact Resolution Priority

Chime uses a multi-tiered approach to resolve contact names:

1. **Local contacts** (\`~/.chime/contacts/\`) - Fastest
2. **macOS Contacts app** (via AppleScript) - Cached for performance
3. **System AddressBook database** - Fallback

## Architecture

### Three-Layer Design

1. **Models Layer** (\`internal/models/\`)
   - Data structures for chats, messages, and contacts

2. **Data Layer** (\`internal/imessage/\`, \`internal/contacts/\`)
   - Read-only SQLite access to iMessage database
   - AppleScript integration for sending messages
   - YAML-based contact storage and retrieval

3. **UI Layer** (\`internal/ui/\`)
   - Bubble Tea components for interactive TUI
   - Menu, conversations, messages, contacts, and forms

### Key Technical Details

- **Database**: Read-only access to \`~/Library/Messages/chat.db\`
- **Message Sending**: AppleScript integration with Messages.app
- **Contact Caching**: Thread-safe in-memory cache with \`sync.RWMutex\`
- **Async Operations**: Contact lookups run in background goroutines
- **Live Updates**: UI refreshes as contact names are resolved

## How It Works

### Reading Messages

Chime reads from your iMessage SQLite database at \`~/Library/Messages/chat.db\`. All database access is read-only to ensure safety.

### Sending Messages

Messages are sent via AppleScript commands to Messages.app. For group chats, Chime tries multiple strategies:
1. Send by chat ID (most reliable)
2. Send by group chat name
3. Create new chat with participants

### Contact Name Resolution

When displaying a phone number or email:
1. Check local \`~/.chime/contacts/\` first (instant)
2. Query macOS Contacts app via AppleScript (cached)
3. Fall back to system AddressBook database

Names load asynchronously and update the UI live as they're found.

## Development

### Project Structure

\`\`\`
chime/
├── main.go                    # Entry point
├── internal/
│   ├── contacts/              # Contact storage & retrieval
│   │   └── contacts.go
│   ├── imessage/              # iMessage integration
│   │   ├── database.go        # Read messages from SQLite
│   │   └── send.go            # Send via AppleScript
│   ├── models/                # Data models
│   │   └── types.go
│   └── ui/                    # Bubble Tea UI components
│       ├── menu.go            # Main menu
│       ├── conversations.go   # Conversation list
│       ├── messages.go        # Message thread
│       ├── contacts_list.go   # Contact list
│       ├── contact_form.go    # Add/edit contact form
│       └── styles.go          # Lipgloss styles
├── go.mod
└── README.md
\`\`\`

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
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

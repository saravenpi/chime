# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

chime is a terminal-based iMessage client for macOS built with Go and Bubble Tea. It reads from the iMessage SQLite database and uses AppleScript to send messages through Messages.app.

## Build and Run

```bash
# Build the application
go build -o chime

# Run directly
./chime

# Run with go
go run main.go

# Install to PATH
go install github.com/saravenpi/chime@latest
```

## Architecture

### Three-Layer Architecture

1. **Models Layer** (`internal/models/types.go`)
   - `Chat`: Represents a conversation with participants, display names, and last message info
   - `Message`: Individual message with text, sender, timestamp, and optional attachments
   - `ViewMode`: Enum for UI state management

2. **Data Layer** (`internal/imessage/`)
   - `database.go`: Read-only SQLite access to `~/Library/Messages/chat.db`
     - `GetChats()`: Fetches all conversations with last message preview
     - `GetMessages(chatID)`: Fetches message history for a conversation
     - `GetContactName()`: Resolves phone numbers to contact names from Address Book
     - `extractTextFromAttributedBody()`: Parses binary attributedBody format (macOS Ventura+)
   - `send.go`: AppleScript-based message sending
     - `SendMessageToChat()`: Routes to individual or group message sending
     - `sendIndividualMessage()`: Sends to single recipient via buddy lookup
     - `sendGroupMessage()`: Sends to multiple participants by creating text chat
     - `sendGroupMessageByChatID()`: Sends using chat ID reference

3. **UI Layer** (`internal/ui/`)
   - `conversations.go`: List view of all chats using Bubble Tea list component
   - `messages.go`: Message thread view with viewport and textarea for composition
   - `styles.go`: Lipgloss styling definitions
   - Uses Bubble Tea's Elm Architecture (Model-Update-View pattern)

### Key Technical Details

**Database Schema Access**:
- `chat` table: conversation metadata
- `message` table: message content (text or attributedBody blob)
- `handle` table: contact identifiers (phone/email)
- `chat_message_join`: links messages to conversations
- `chat_handle_join`: links participants to group chats
- `attachment` table: file attachment paths

**Message Sending Strategy**:
- Individual messages: Use `buddy` object via service lookup
- Group messages: Three fallback approaches:
  1. By chat name (for named group chats)
  2. By chat ID (using `text chat id` reference)
  3. By participants list (creates new chat with properties)
- All AppleScript calls return "SUCCESS" or "ERROR:" prefixed strings

**AttributedBody Parsing**:
- macOS Ventura+ stores messages as binary NSAttributedString
- Parser searches for "NSString" markers in binary data
- Extracts text length (1-byte or 3-byte length encoding)
- Validates UTF-8 and concatenates text segments

## Group Chat Support

Group chats are identified by `chat_identifier` starting with "chat" (e.g., "chat936183701855695764").

Current implementation attempts three sending strategies:
1. `sendGroupMessageByName()`: Finds chat by display name
2. `sendGroupMessageByChatID()`: Uses chat ID with "iMessage;+;" prefix conversion
3. `sendGroupMessage()`: Creates new text chat with participant list

If experiencing "Can't get text chat" errors, the issue is likely in the AppleScript chat reference lookup. The database format uses "any;+;chatXXX" but AppleScript expects "iMessage;+;chatXXX".

## Common Development Tasks

Test database connectivity:
```bash
sqlite3 ~/Library/Messages/chat.db "SELECT COUNT(*) FROM chat"
```

Check AppleScript manually:
```bash
osascript -e 'tell application "Messages" to get name of every text chat'
```

Grant Full Disk Access to terminal:
System Preferences → Security & Privacy → Privacy → Full Disk Access → Add Terminal/iTerm

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): TUI framework with Elm Architecture
- [Bubbles](https://github.com/charmbracelet/bubbles): Pre-built TUI components (list, textarea, viewport)
- [Lipgloss](https://github.com/charmbracelet/lipgloss): Terminal styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3): CGO-based SQLite driver
- [reflow](https://github.com/muesli/reflow): Text wrapping utilities

## Limitations

- macOS only (iMessage database is platform-specific)
- Messages.app must be running and signed in for sending
- Cannot edit/delete existing messages (read-only database access)
- Cannot create new group chats from scratch (only reply to existing)
- Group chat sending has AppleScript reference issues with chat ID lookup

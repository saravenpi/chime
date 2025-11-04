package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saravenpi/chime/internal/ui"
)

const version = "1.0.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("Pirouette v%s\n", version)
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	initialModel := ui.NewMenuModel()
	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `Chime - Terminal iMessage Client

Usage:
  chime              Start the iMessage client
  chime version      Show version information
  chime help         Show this help message

Navigation:
  ↑/↓ or j/k        Navigate lists
  Enter             Select/Open item
  ESC               Go back
  q                 Quit from current view
  ctrl+c            Force quit

Menu:
  💬 Conversations  View and send messages
  👥 Contacts       Manage contacts

Contacts:
  n or a            Add new contact
  enter             Edit contact
  d                 Delete contact
  ctrl+s            Save contact (while editing)

Conversations:
  /                 Search conversations
  r                 Refresh conversation list

Messages:
  n or c            Compose new message
  r                 Refresh messages
  ctrl+s            Send message (while composing)
  ↑/↓ or j/k        Scroll messages

Contact Storage:
  Contacts are stored in ~/.chime/contacts/ as YAML files
  Each contact has a name, phone numbers, and email addresses

Notes:
  - This app reads from your iMessage database (read-only)
  - Sending messages uses AppleScript to interact with Messages.app
  - Make sure Messages.app is running on your Mac
`
	fmt.Print(help)
}

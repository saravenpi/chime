package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saravenpi/chime/internal/imessage"
	"github.com/saravenpi/chime/internal/models"
)

type chatItem struct {
	chat  models.Chat
	index int
}

type chatsFetchedMsg struct {
	chats []models.Chat
	err   error
}

func (i chatItem) Title() string {
	return i.chat.DisplayName
}

func (i chatItem) Description() string {
	timeAgo := formatTimeAgo(i.chat.LastTime)
	preview := i.chat.LastMessage
	if len(preview) > 50 {
		preview = preview[:47] + "..."
	}
	return fmt.Sprintf("%s • %s", timeAgo, preview)
}

func (i chatItem) FilterValue() string {
	return i.chat.DisplayName
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	duration := now.Sub(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < 2*time.Minute {
		return "1 min ago"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	}
	if duration < 2*time.Hour {
		return "1h ago"
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	}
	if duration < 48*time.Hour {
		return "yesterday"
	}
	if duration < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
	return t.Format("Jan 2")
}

type ConversationsModel struct {
	chats         []models.Chat
	list          list.Model
	loading       bool
	err           error
	spinner       spinner.Model
	windowWidth   int
	windowHeight  int
	selectedIndex int
}

func NewConversationsModel() ConversationsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = statusStyle

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("5")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("8"))

	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "Conversations"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return ConversationsModel{
		list:         l,
		loading:      true,
		spinner:      s,
		windowWidth:  80,
		windowHeight: 30,
	}
}

func (m ConversationsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchChatsCmd())
}

func (m ConversationsModel) fetchChatsCmd() tea.Cmd {
	return func() tea.Msg {
		chats, err := imessage.GetChats()
		return chatsFetchedMsg{chats: chats, err: err}
	}
}

func (m ConversationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case chatsFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.chats = msg.chats
		items := make([]list.Item, len(m.chats))
		for i, chat := range m.chats {
			items[i] = chatItem{chat: chat, index: i}
		}
		m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("Conversations - %d chats", len(m.chats))
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "esc" {
			menuModel := NewMenuModel()
			return menuModel, menuModel.Init()
		}

		if msg.String() == "r" && !m.loading {
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchChatsCmd())
		}

		if msg.String() == "enter" && len(m.chats) > 0 && !m.loading {
			if item, ok := m.list.SelectedItem().(chatItem); ok {
				messagesModel := NewMessagesModel(item.chat)
				return messagesModel, messagesModel.Init()
			}
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ConversationsModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading conversations...\n", m.spinner.View())
	}

	if m.err != nil {
		s := titleStyle.Render("Conversations") + "\n\n"
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		s += helpStyle.Render("Make sure you have access to Messages.app database") + "\n"
		s += helpStyle.Render("q: quit")
		return s
	}

	if len(m.chats) == 0 {
		s := titleStyle.Render("Conversations") + "\n\n"
		s += normalStyle.Render("  No conversations found.") + "\n"
		s += "\n" + helpStyle.Render("r: refresh • q: quit")
		return s
	}

	s := m.list.View() + "\n"
	s += helpStyle.Render("↑↓/jk: navigate • enter: open • /: search • r: refresh • esc: back • q: quit")

	return s
}

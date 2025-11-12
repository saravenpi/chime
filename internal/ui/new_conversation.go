package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saravenpi/chime/internal/imessage"
	"github.com/saravenpi/chime/internal/models"
)

type NewConversationModel struct {
	recipientInput textinput.Model
	messageInput   textinput.Model
	focusIndex     int
	windowWidth    int
	windowHeight   int
	showUnreadOnly bool
	err            error
}

func NewNewConversationModel(showUnreadOnly bool) NewConversationModel {
	recipientInput := textinput.New()
	recipientInput.Placeholder = "Phone number or email (e.g., +1234567890 or user@example.com)"
	recipientInput.Focus()
	recipientInput.CharLimit = 100
	recipientInput.Width = 60

	messageInput := textinput.New()
	messageInput.Placeholder = "Type your message here..."
	messageInput.CharLimit = 1000
	messageInput.Width = 60

	return NewConversationModel{
		recipientInput: recipientInput,
		messageInput:   messageInput,
		focusIndex:     0,
		showUnreadOnly: showUnreadOnly,
	}
}

func (m NewConversationModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m NewConversationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.recipientInput.Width = msg.Width - 20
		m.messageInput.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			conversationsModel := NewConversationsModel()
			conversationsModel.showUnreadOnly = m.showUnreadOnly
			if m.windowWidth > 0 {
				updatedModel, _ := conversationsModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				conversationsModel = updatedModel.(ConversationsModel)
			}
			return conversationsModel, conversationsModel.Init()

		case "tab", "shift+tab":
			if msg.String() == "tab" {
				m.focusIndex = (m.focusIndex + 1) % 2
			} else {
				m.focusIndex = (m.focusIndex - 1 + 2) % 2
			}

			if m.focusIndex == 0 {
				m.recipientInput.Focus()
				m.messageInput.Blur()
			} else {
				m.recipientInput.Blur()
				m.messageInput.Focus()
			}
			return m, nil

		case "enter":
			if m.recipientInput.Value() == "" {
				m.err = nil
				return m, nil
			}

			if m.messageInput.Value() == "" {
				m.err = nil
				return m, nil
			}

			recipient := m.recipientInput.Value()
			message := m.messageInput.Value()

			err := imessage.SendMessage(recipient, message)
			if err != nil {
				m.err = err
				return m, nil
			}

			chat := models.Chat{
				ChatID:      recipient,
				DisplayName: recipient,
				IsGroup:     false,
			}

			messagesModel := NewMessagesModel(chat, m.showUnreadOnly)
			if m.windowWidth > 0 {
				updatedModel, _ := messagesModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				messagesModel = updatedModel.(MessagesModel)
			}
			return messagesModel, messagesModel.Init()
		}
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.recipientInput, cmd = m.recipientInput.Update(msg)
	} else {
		m.messageInput, cmd = m.messageInput.Update(msg)
	}
	return m, cmd
}

func (m NewConversationModel) View() string {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("5"))

	title := titleStyle.Render("New Conversation")

	recipientLabel := "Recipient:"
	if m.focusIndex == 0 {
		recipientLabel = "> " + recipientLabel
	} else {
		recipientLabel = "  " + recipientLabel
	}

	messageLabel := "Message:"
	if m.focusIndex == 1 {
		messageLabel = "> " + messageLabel
	} else {
		messageLabel = "  " + messageLabel
	}

	content := title + "\n\n"
	content += style.Render(
		recipientLabel + "\n" +
			m.recipientInput.View() + "\n\n" +
			messageLabel + "\n" +
			m.messageInput.View(),
	)

	if m.err != nil {
		content += "\n\n" + errorStyle.Render("Error: "+m.err.Error())
	}

	content += "\n\n" + helpStyle.Render("tab: switch field • enter: send • esc: back • q: quit")

	return content
}

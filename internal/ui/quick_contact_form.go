package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saravenpi/chime/internal/contacts"
	"github.com/saravenpi/chime/internal/imessage"
	"github.com/saravenpi/chime/internal/models"
)

type quickContactSavedMsg struct {
	success bool
	err     error
	chat    models.Chat
}

type QuickContactFormModel struct {
	chat           models.Chat
	identifier     string
	nameInput      textinput.Model
	err            error
	windowWidth    int
	windowHeight   int
	showUnreadOnly bool
}

func NewQuickContactFormModel(chat models.Chat, identifier string, showUnreadOnly bool) QuickContactFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Contact Name"
	nameInput.Focus()
	nameInput.CharLimit = 100
	nameInput.Width = 50
	nameInput.SetValue(chat.DisplayName)

	return QuickContactFormModel{
		chat:           chat,
		identifier:     identifier,
		nameInput:      nameInput,
		showUnreadOnly: showUnreadOnly,
	}
}

func (m QuickContactFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m QuickContactFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "esc" {
			messagesModel := NewMessagesModel(m.chat, m.showUnreadOnly)
			if m.windowWidth > 0 {
				updatedModel, _ := messagesModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				messagesModel = updatedModel.(MessagesModel)
			}
			return messagesModel, messagesModel.Init()
		}

		if msg.String() == "enter" || msg.String() == "ctrl+s" {
			return m, m.saveContact()
		}

	case quickContactSavedMsg:
		if msg.success {
			messagesModel := NewMessagesModel(msg.chat, m.showUnreadOnly)
			if m.windowWidth > 0 {
				updatedModel, _ := messagesModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				messagesModel = updatedModel.(MessagesModel)
			}
			return messagesModel, messagesModel.Init()
		}
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m QuickContactFormModel) saveContact() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return quickContactSavedMsg{success: false, err: fmt.Errorf("name is required"), chat: m.chat}
		}

		contact := contacts.Contact{
			Name: name,
		}

		if strings.Contains(m.identifier, "@") {
			contact.Emails = []string{m.identifier}
		} else {
			contact.PhoneNumbers = []string{m.identifier}
		}

		err := contacts.SaveContact(contact)
		if err != nil {
			return quickContactSavedMsg{success: false, err: err, chat: m.chat}
		}

		updatedChats, err := imessage.GetChats()
		if err != nil {
			return quickContactSavedMsg{success: false, err: err, chat: m.chat}
		}

		for _, chat := range updatedChats {
			if chat.ROWID == m.chat.ROWID {
				return quickContactSavedMsg{success: true, err: nil, chat: chat}
			}
		}

		return quickContactSavedMsg{success: true, err: nil, chat: m.chat}
	}
}

func (m QuickContactFormModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Add Contact") + "\n\n")

	identifierType := "Phone"
	if strings.Contains(m.identifier, "@") {
		identifierType = "Email"
	}

	b.WriteString(normalStyle.Render(fmt.Sprintf("%s: %s", identifierType, m.identifier)) + "\n\n")

	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	b.WriteString(focusedStyle.Render("Name:") + "\n")
	b.WriteString(m.nameInput.View() + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	b.WriteString(helpStyle.Render("enter/ctrl+s: save • esc: cancel"))

	return b.String()
}

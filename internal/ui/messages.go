package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/saravenpi/chime/internal/contacts"
	"github.com/saravenpi/chime/internal/imessage"
	"github.com/saravenpi/chime/internal/models"
)

type messagesFetchedMsg struct {
	messages []models.Message
	err      error
}

type messageSentMsg struct {
	err error
}

type MessagesModel struct {
	chat           models.Chat
	messages       []models.Message
	viewport       viewport.Model
	textarea       textarea.Model
	loading        bool
	sending        bool
	composing      bool
	err            error
	spinner        spinner.Model
	windowWidth    int
	windowHeight   int
	viewportReady  bool
	showUnreadOnly bool
}

func NewMessagesModel(chat models.Chat, showUnreadOnly bool) MessagesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = statusStyle

	vp := viewport.New(80, 20)
	vp.HighPerformanceRendering = false

	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.CharLimit = 1000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	return MessagesModel{
		chat:           chat,
		viewport:       vp,
		textarea:       ta,
		loading:        true,
		spinner:        s,
		windowWidth:    80,
		windowHeight:   30,
		viewportReady:  true,
		showUnreadOnly: showUnreadOnly,
	}
}

func (m MessagesModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchMessagesCmd())
}

func (m MessagesModel) fetchMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		messages, err := imessage.GetMessages(m.chat.ROWID)
		if err != nil {
			return messagesFetchedMsg{messages: nil, err: err}
		}

		if err := imessage.MarkChatAsRead(m.chat.ROWID); err != nil {
		}

		return messagesFetchedMsg{messages: messages, err: nil}
	}
}

func (m MessagesModel) sendMessageCmd(message string) tea.Cmd {
	return func() tea.Msg {
		err := imessage.SendMessageToChat(m.chat, message)
		return messageSentMsg{err: err}
	}
}

func (m MessagesModel) canAddContact() bool {
	if m.chat.IsGroup {
		return false
	}

	if len(m.chat.Participants) == 0 {
		return false
	}

	identifier := m.chat.Participants[0]
	existingName := contacts.FindContactByIdentifier(identifier)
	return existingName == ""
}

func (m MessagesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

		headerHeight := 6
		textareaHeight := 5
		helpHeight := 2
		availableHeight := msg.Height - headerHeight - helpHeight

		if m.composing {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = availableHeight - textareaHeight
			m.textarea.SetWidth(msg.Width - 4)
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = availableHeight
		}

		m.updateViewportContent()
		return m, nil

	case messagesFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.messages = msg.messages
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case messageSentMsg:
		m.sending = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.textarea.Reset()
		m.composing = false
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return m.fetchMessagesCmd()()
		}))

	case spinner.TickMsg:
		if m.loading || m.sending {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "esc" {
			if m.composing {
				m.composing = false
				m.textarea.Reset()
				m.textarea.Blur()
				m.err = nil
				return m, nil
			}
			convModel := NewConversationsModel()
			convModel.showUnreadOnly = m.showUnreadOnly
			if m.windowWidth > 0 {
				updatedModel, cmd := convModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				convModel = updatedModel.(ConversationsModel)
				return convModel, tea.Batch(convModel.Init(), cmd)
			}
			return convModel, convModel.Init()
		}

		if m.composing {
			switch msg.String() {
			case "ctrl+s":
				messageText := strings.TrimSpace(m.textarea.Value())
				if messageText != "" {
					m.sending = true
					m.composing = false
					m.textarea.Blur()
					return m, tea.Batch(
						m.spinner.Tick,
						m.sendMessageCmd(messageText),
					)
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}

		if m.loading || m.sending {
			return m, nil
		}

		switch msg.String() {
		case "n", "c":
			m.composing = true
			m.textarea.Focus()
			return m, textarea.Blink

		case "a":
			if m.canAddContact() && len(m.chat.Participants) > 0 {
				quickForm := NewQuickContactFormModel(m.chat, m.chat.Participants[0], m.showUnreadOnly)
				if m.windowWidth > 0 {
					updatedModel, _ := quickForm.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
					quickForm = updatedModel.(QuickContactFormModel)
				}
				return quickForm, quickForm.Init()
			}
			return m, nil

		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.fetchMessagesCmd())

		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *MessagesModel) updateViewportContent() {
	if !m.viewportReady || len(m.messages) == 0 {
		return
	}

	var content strings.Builder
	wrapWidth := m.viewport.Width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	for i, message := range m.messages {
		if i > 0 {
			content.WriteString("\n")
		}

		timestamp := message.Date.Format("3:04 PM")

		if message.IsFromMe {
			sender := "You"
			header := messageHeaderStyle.Render(fmt.Sprintf("%s • %s", sender, timestamp))
			content.WriteString(lipgloss.NewStyle().Align(lipgloss.Right).Width(wrapWidth).Render(header) + "\n")

			if message.Text != "" {
				wrappedText := wordwrap.String(message.Text, wrapWidth-10)
				styledText := messageFromMeStyle.Render(wrappedText)
				content.WriteString(lipgloss.NewStyle().Align(lipgloss.Right).Width(wrapWidth).Render(styledText) + "\n")
			}

			if message.AttachmentPath != "" {
				attachmentText := fmt.Sprintf("📎 [Attachment: %s]", message.AttachmentPath)
				styledAttachment := messageHeaderStyle.Render(attachmentText)
				content.WriteString(lipgloss.NewStyle().Align(lipgloss.Right).Width(wrapWidth).Render(styledAttachment) + "\n")
			}
		} else {
			sender := message.Handle
			if sender == "" {
				sender = "Unknown"
			}
			header := messageHeaderStyle.Render(fmt.Sprintf("%s • %s", sender, timestamp))
			content.WriteString(header + "\n")

			if message.Text != "" {
				wrappedText := wordwrap.String(message.Text, wrapWidth-10)
				styledText := messageFromOtherStyle.Render(wrappedText)
				content.WriteString(styledText + "\n")
			}

			if message.AttachmentPath != "" {
				attachmentText := fmt.Sprintf("📎 [Attachment: %s]", message.AttachmentPath)
				content.WriteString(messageHeaderStyle.Render(attachmentText) + "\n")
			}
		}
	}

	m.viewport.SetContent(content.String())
}

func (m MessagesModel) View() string {
	if m.loading && len(m.messages) == 0 {
		return fmt.Sprintf("\n  %s Loading messages...\n", m.spinner.View())
	}

	s := titleStyle.Render(fmt.Sprintf("💬 %s", m.chat.DisplayName)) + "\n\n"

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	if m.sending {
		s += fmt.Sprintf("  %s Sending message...\n", m.spinner.View())
	} else if len(m.messages) == 0 && !m.loading {
		s += normalStyle.Render("  No messages in this conversation.") + "\n"
	} else {
		s += m.viewport.View() + "\n"
	}

	if m.composing {
		s += "\n" + inputStyle.Render("New Message:") + "\n"
		s += m.textarea.View() + "\n"
		s += helpStyle.Render("ctrl+s: send • esc: cancel")
	} else {
		scrollPercent := int(m.viewport.ScrollPercent() * 100)
		helpText := fmt.Sprintf("↑↓/jk: scroll • n: new message • r: refresh • esc: back • q: quit • %d%%", scrollPercent)
		if m.canAddContact() {
			helpText = fmt.Sprintf("↑↓/jk: scroll • n: new message • a: add contact • r: refresh • esc: back • q: quit • %d%%", scrollPercent)
		}
		s += "\n" + helpStyle.Render(helpText)
	}

	return s
}

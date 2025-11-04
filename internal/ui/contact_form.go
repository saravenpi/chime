package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saravenpi/chime/internal/contacts"
)

type contactSavedMsg struct {
	success bool
	err     error
}

type ContactFormModel struct {
	originalContact *contacts.Contact
	nameInput       textinput.Model
	phoneInputs     []textinput.Model
	emailInputs     []textinput.Model
	focusIndex      int
	err             error
	saved           bool
}

// NewContactFormModel creates a form for adding or editing a contact.
func NewContactFormModel(contact *contacts.Contact) ContactFormModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Contact Name"
	nameInput.Focus()
	nameInput.CharLimit = 100
	nameInput.Width = 50

	phoneInputs := make([]textinput.Model, 3)
	for i := range phoneInputs {
		phoneInputs[i] = textinput.New()
		phoneInputs[i].Placeholder = fmt.Sprintf("Phone Number %d (optional)", i+1)
		phoneInputs[i].CharLimit = 20
		phoneInputs[i].Width = 50
	}

	emailInputs := make([]textinput.Model, 3)
	for i := range emailInputs {
		emailInputs[i] = textinput.New()
		emailInputs[i].Placeholder = fmt.Sprintf("Email %d (optional)", i+1)
		emailInputs[i].CharLimit = 100
		emailInputs[i].Width = 50
	}

	m := ContactFormModel{
		originalContact: contact,
		nameInput:       nameInput,
		phoneInputs:     phoneInputs,
		emailInputs:     emailInputs,
		focusIndex:      0,
	}

	if contact != nil {
		m.nameInput.SetValue(contact.Name)
		for i, phone := range contact.PhoneNumbers {
			if i < len(m.phoneInputs) {
				m.phoneInputs[i].SetValue(phone)
			}
		}
		for i, email := range contact.Emails {
			if i < len(m.emailInputs) {
				m.emailInputs[i].SetValue(email)
			}
		}
	}

	return m
}

func (m ContactFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ContactFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "esc" {
			contactsModel := NewContactsListModel()
			return contactsModel, contactsModel.Init()
		}

		if msg.String() == "tab" || msg.String() == "shift+tab" || msg.String() == "down" || msg.String() == "up" {
			totalInputs := 1 + len(m.phoneInputs) + len(m.emailInputs)

			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = totalInputs - 1
				}
			} else {
				m.focusIndex++
				if m.focusIndex >= totalInputs {
					m.focusIndex = 0
				}
			}

			m.updateFocus()
			return m, nil
		}

		if msg.String() == "ctrl+s" {
			return m, m.saveContact()
		}

	case contactSavedMsg:
		if msg.success {
			contactsModel := NewContactsListModel()
			return contactsModel, contactsModel.Init()
		}
		m.err = msg.err
		return m, nil
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *ContactFormModel) updateFocus() {
	m.nameInput.Blur()
	for i := range m.phoneInputs {
		m.phoneInputs[i].Blur()
	}
	for i := range m.emailInputs {
		m.emailInputs[i].Blur()
	}

	if m.focusIndex == 0 {
		m.nameInput.Focus()
	} else if m.focusIndex <= len(m.phoneInputs) {
		m.phoneInputs[m.focusIndex-1].Focus()
	} else {
		emailIndex := m.focusIndex - 1 - len(m.phoneInputs)
		if emailIndex < len(m.emailInputs) {
			m.emailInputs[emailIndex].Focus()
		}
	}
}

func (m *ContactFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0)

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	cmds = append(cmds, cmd)

	for i := range m.phoneInputs {
		m.phoneInputs[i], cmd = m.phoneInputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	for i := range m.emailInputs {
		m.emailInputs[i], cmd = m.emailInputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m ContactFormModel) saveContact() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return contactSavedMsg{success: false, err: fmt.Errorf("name is required")}
		}

		phoneNumbers := []string{}
		for _, input := range m.phoneInputs {
			phone := strings.TrimSpace(input.Value())
			if phone != "" {
				phoneNumbers = append(phoneNumbers, phone)
			}
		}

		emails := []string{}
		for _, input := range m.emailInputs {
			email := strings.TrimSpace(input.Value())
			if email != "" {
				emails = append(emails, email)
			}
		}

		if len(phoneNumbers) == 0 && len(emails) == 0 {
			return contactSavedMsg{success: false, err: fmt.Errorf("at least one phone number or email is required")}
		}

		if m.originalContact != nil && m.originalContact.Name != name {
			if err := contacts.DeleteContact(m.originalContact.Name); err != nil {
				return contactSavedMsg{success: false, err: fmt.Errorf("failed to delete old contact: %w", err)}
			}
		}

		contact := contacts.Contact{
			Name:         name,
			PhoneNumbers: phoneNumbers,
			Emails:       emails,
		}

		if err := contacts.SaveContact(contact); err != nil {
			return contactSavedMsg{success: false, err: err}
		}

		return contactSavedMsg{success: true, err: nil}
	}
}

func (m ContactFormModel) View() string {
	var b strings.Builder

	title := "Add Contact"
	if m.originalContact != nil {
		title = "Edit Contact"
	}

	b.WriteString(titleStyle.Render(title) + "\n\n")

	focusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	blurredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	renderInput := func(input textinput.Model, label string, focused bool) {
		style := blurredStyle
		if focused {
			style = focusedStyle
		}
		b.WriteString(style.Render(label) + "\n")
		b.WriteString(input.View() + "\n\n")
	}

	renderInput(m.nameInput, "Name (required):", m.focusIndex == 0)

	b.WriteString(normalStyle.Render("Phone Numbers:") + "\n")
	for i, input := range m.phoneInputs {
		renderInput(input, fmt.Sprintf("  Phone %d:", i+1), m.focusIndex == i+1)
	}

	b.WriteString(normalStyle.Render("Email Addresses:") + "\n")
	for i, input := range m.emailInputs {
		emailIndex := i + 1 + len(m.phoneInputs)
		renderInput(input, fmt.Sprintf("  Email %d:", i+1), m.focusIndex == emailIndex)
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}

	b.WriteString(helpStyle.Render("tab/↑↓: navigate • ctrl+s: save • esc: cancel"))

	return b.String()
}

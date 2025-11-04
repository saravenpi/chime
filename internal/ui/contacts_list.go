package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saravenpi/chime/internal/contacts"
)

type contactItem struct {
	contact contacts.Contact
}

func (i contactItem) FilterValue() string { return i.contact.Name }
func (i contactItem) Title() string       { return i.contact.Name }
func (i contactItem) Description() string {
	desc := ""
	if len(i.contact.PhoneNumbers) > 0 {
		desc += i.contact.PhoneNumbers[0]
	}
	if len(i.contact.Emails) > 0 {
		if desc != "" {
			desc += " • "
		}
		desc += i.contact.Emails[0]
	}
	return desc
}

type contactsLoadedMsg struct {
	contacts []contacts.Contact
	err      error
}

type ContactsListModel struct {
	list                list.Model
	contacts            []contacts.Contact
	loading             bool
	err                 error
	windowWidth         int
	windowHeight        int
	confirmDelete       bool
	contactToDelete     *contacts.Contact
}

// NewContactsListModel creates a new contacts list view.
func NewContactsListModel() ContactsListModel {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("5")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("8"))

	l := list.New([]list.Item{}, delegate, 80, 20)
	l.Title = "Contacts"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	return ContactsListModel{
		list:         l,
		loading:      true,
		windowWidth:  80,
		windowHeight: 30,
	}
}

func (m ContactsListModel) Init() tea.Cmd {
	return m.loadContactsCmd()
}

func (m ContactsListModel) loadContactsCmd() tea.Cmd {
	return func() tea.Msg {
		contacts, err := contacts.ListContacts()
		if err != nil {
			return contactsLoadedMsg{contacts: nil, err: err}
		}
		return contactsLoadedMsg{contacts: contacts, err: nil}
	}
}

func (m ContactsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case contactsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.contacts = msg.contacts
		items := make([]list.Item, len(m.contacts))
		for i, contact := range m.contacts {
			items[i] = contactItem{contact: contact}
		}
		m.list.SetItems(items)
		m.list.Title = fmt.Sprintf("Contacts - %d total", len(m.contacts))
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.confirmDelete {
			if msg.String() == "y" || msg.String() == "Y" {
				if m.contactToDelete != nil {
					if err := contacts.DeleteContact(m.contactToDelete.Name); err == nil {
						m.confirmDelete = false
						m.contactToDelete = nil
						m.loading = true
						return m, m.loadContactsCmd()
					}
				}
				m.confirmDelete = false
				m.contactToDelete = nil
				return m, nil
			}
			if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" {
				m.confirmDelete = false
				m.contactToDelete = nil
				return m, nil
			}
			return m, nil
		}

		if msg.String() == "esc" || msg.String() == "q" {
			menuModel := NewMenuModel()
			if m.windowWidth > 0 {
				updatedModel, _ := menuModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				menuModel = updatedModel.(MenuModel)
			}
			return menuModel, menuModel.Init()
		}

		if msg.String() == "n" || msg.String() == "a" {
			formModel := NewContactFormModel(nil)
			if m.windowWidth > 0 {
				updatedModel, _ := formModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
				formModel = updatedModel.(ContactFormModel)
			}
			return formModel, formModel.Init()
		}

		if msg.String() == "r" {
			m.loading = true
			return m, m.loadContactsCmd()
		}

		if msg.String() == "enter" && len(m.contacts) > 0 {
			if item, ok := m.list.SelectedItem().(contactItem); ok {
				formModel := NewContactFormModel(&item.contact)
				if m.windowWidth > 0 {
					updatedModel, _ := formModel.Update(tea.WindowSizeMsg{Width: m.windowWidth, Height: m.windowHeight})
					formModel = updatedModel.(ContactFormModel)
				}
				return formModel, formModel.Init()
			}
		}

		if msg.String() == "d" || msg.String() == "delete" {
			if item, ok := m.list.SelectedItem().(contactItem); ok {
				m.confirmDelete = true
				contactCopy := item.contact
				m.contactToDelete = &contactCopy
				return m, nil
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ContactsListModel) View() string {
	if m.confirmDelete && m.contactToDelete != nil {
		s := titleStyle.Render("Delete Contact") + "\n\n"
		s += normalStyle.Render(fmt.Sprintf("Are you sure you want to delete '%s'?", m.contactToDelete.Name)) + "\n\n"
		s += errorStyle.Render("This action cannot be undone.") + "\n\n"
		s += helpStyle.Render("y: confirm delete • n/esc: cancel")
		return s
	}

	if m.loading {
		return "\n  Loading contacts...\n"
	}

	if m.err != nil {
		s := titleStyle.Render("Contacts") + "\n\n"
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		s += helpStyle.Render("esc: back to menu • q: quit")
		return s
	}

	if len(m.contacts) == 0 {
		s := titleStyle.Render("Contacts") + "\n\n"
		s += normalStyle.Render("  No contacts found. Press 'n' to add a contact.") + "\n"
		s += "\n" + helpStyle.Render("n: new contact • esc: back • q: quit")
		return s
	}

	s := m.list.View() + "\n"
	s += helpStyle.Render("↑↓/jk: navigate • enter: edit • n: new • d: delete • /: search • r: refresh • esc: back • q: quit")

	return s
}

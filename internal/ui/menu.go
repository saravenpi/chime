package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	title string
	desc  string
}

func (i menuItem) FilterValue() string { return i.title }
func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }

type MenuModel struct {
	list         list.Model
	windowWidth  int
	windowHeight int
}

// NewMenuModel creates the main menu with Conversations and Contacts options.
func NewMenuModel() MenuModel {
	items := []list.Item{
		menuItem{title: "💬 Conversations", desc: "View and send messages"},
		menuItem{title: "👥 Contacts", desc: "Manage your contacts"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("5")).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("8"))

	l := list.New(items, delegate, 80, 14)
	l.Title = "Chime - iMessage Terminal Client"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return MenuModel{
		list:         l,
		windowWidth:  80,
		windowHeight: 30,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		if msg.String() == "enter" {
			selectedItem, ok := m.list.SelectedItem().(menuItem)
			if !ok {
				return m, nil
			}

			if selectedItem.title == "💬 Conversations" {
				conversationsModel := NewConversationsModel()
				return conversationsModel, conversationsModel.Init()
			} else if selectedItem.title == "👥 Contacts" {
				contactsModel := NewContactsListModel()
				return contactsModel, contactsModel.Init()
			}
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MenuModel) View() string {
	s := m.list.View() + "\n"
	s += helpStyle.Render("↑↓/jk: navigate • enter: select • q: quit")
	return s
}

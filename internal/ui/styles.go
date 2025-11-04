package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213")).
		MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("213"))

	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("117"))

	messageFromMeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("111")).
		Align(lipgloss.Right)

	messageFromOtherStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("120"))

	messageHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("117")).
		Bold(true)
)

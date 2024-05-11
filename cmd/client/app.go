package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zoujiaqing/quic-chat/internal/chat"
)

type clientMsg struct {
	message chat.Message
	err     error
}

type app struct {
	viewport viewport.Model
	textarea textarea.Model
	lines    []string
	send     func(text string) error
	messages <-chan chat.Message
	errs     <-chan error
}

func createApp(send func(text string) error, messages <-chan chat.Message, errs <-chan error) *app {
	vp := viewport.New(30, 10)

	ta := textarea.New()
	ta.Placeholder = "Message"
	ta.Focus()
	ta.SetWidth(30)
	ta.SetHeight(1)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	return &app{
		viewport: vp,
		textarea: ta,
		lines:    []string{},
		send:     send,
		messages: messages,
		errs:     errs,
	}
}

func (a *app) Init() tea.Cmd {
	return a.waitForMessageOrError()
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		vpCmd tea.Cmd
		taCmd tea.Cmd
	)
	a.viewport, vpCmd = a.viewport.Update(msg)
	a.textarea, taCmd = a.textarea.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return a, tea.Quit
		case tea.KeyEnter:
			if err := a.send(a.textarea.Value()); err != nil {
				return a, tea.Quit
			}
			a.textarea.Reset()
		}
	case clientMsg:
		if msg.err != nil {
			return a, tea.Quit
		}
		a.lines = append(a.lines, fmt.Sprintf("[%s]: %s", msg.message.Nickname, msg.message.Text))
		a.viewport.SetContent(strings.Join(a.lines, "\n"))
		a.viewport.GotoBottom()
		return a, a.waitForMessageOrError()
	}

	return a, tea.Batch(vpCmd, taCmd)
}

func (a *app) View() string {
	return fmt.Sprintf("%s\n\n%s\n", a.viewport.View(), a.textarea.View())
}

func (a *app) waitForMessageOrError() tea.Cmd {
	return func() tea.Msg {
		select {
		case message := <-a.messages:
			return clientMsg{message: message}
		case err := <-a.errs:
			return clientMsg{err: err}
		}
	}
}

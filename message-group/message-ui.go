package messagegroup

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/onioncall/squa/common"
	"github.com/onioncall/squa/entities"
	"github.com/onioncall/squa/services"
	"golang.org/x/term"
)

func Execute(groupUuid uuid.UUID) {
	services.Clear()
	go entities.MessagesService()
	p := tea.NewProgram(initialModel(groupUuid))

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func initialModel(groupUuid uuid.UUID) model {
	tw, th, _ := term.GetSize(int(os.Stdout.Fd()))

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.SetWidth(tw-2)
	ta.SetHeight(5)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(tw-2, th-9)
	vp.Style = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	PaddingRight(2).
	PaddingTop(1).
	PaddingLeft(2)

	welcomeMessage := fmt.Sprintf("Welcome to message group %s!\nType a message and press Enter to send.", groupUuid.String())

	vp.SetContent(welcomeMessage)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:      ta,
		messages:      []string{},
		viewport:      vp,
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("62")), //lets do 5 for other chats
		recieverStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		err:           nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			u := entities.User
			u.DeactivateUser();
			log.Fatal("Goodbye!")
			return m, tea.Quit

		case tea.KeyEnter:
			message := entities.DisplayMessage{
				DisplayName:     entities.User.DisplayName,
				MessageContents: m.textarea.Value(),
			}
			
			message.SendMessage()
			m.messages = append(m.messages, m.senderStyle.Render(entities.User.DisplayName+": ")+m.textarea.Value())
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	if len(entities.UnrecievedMessages) > 0 {
		for _, message := range entities.UnrecievedMessages {
			m.messages = append(m.messages, m.recieverStyle.Render(message.DisplayName+": ")+message.MessageContents)
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
		}

		entities.UnrecievedMessages = []entities.DisplayMessage{}
	}

	if len(common.Errorlist) > 0 {
		for _, err := range common.Errorlist {
			m.messages = append(m.messages, m.errorStyle.Render("Error: ")+err.Error())
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.viewport.GotoBottom()
		}

		common.Errorlist = []error{}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n%s",
		m.viewport.View(),
		m.textarea.View(),
		helpStyle(entities.Group.GroupUuid.String()),
	)
}
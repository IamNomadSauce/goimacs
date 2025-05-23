package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	currentDir    string
	files         []os.DirEntry
	selectedIndex int
	focused       string

	filePath      string
	lines         []string
	cursorX       int
	cursorY       int
	viewStart     int
	mode          string
	commandBuffer string

	termWidth     int
	termHeight    int
	explorerWidth int
}

func initialModel() model {
	currentDir, _ := os.Getwd()
	files, _ := os.ReadDir(currentDir)
	return model{
		currentDir:    currentDir,
		files:         files,
		selectedIndex: 0,
		focused:       "explorer",
		mode:          "normal",
		termWidth:     80,
		termHeight:    24,
		explorerWidth: 20,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.explorerWidth = m.termWidth / 5
		return m, nil
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch m.focused {
		case "explorer":
			m, cmd = updateExplorer(m, msg)
		case "editor":
			m, cmd = updateEditor(m, msg)
		}
		return m, cmd
	}
	return m, nil
}

func updateExplorer(m model, msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case "down", "j":
		if m.selectedIndex < len(m.files)-1 {
			m.selectedIndex++
		}
	case "enter":
		selectedFile := m.files[m.selectedIndex]
		if selectedFile.IsDir() {
			m.currentDir = filepath.Join(m.currentDir, selectedFile.Name())
			files, _ := os.ReadDir(m.currentDir)
			m.files = files
			m.selectedIndex = 0
		} else {
			filePath := filepath.Join(m.currentDir, selectedFile.Name())
			content, _ := os.ReadFile(filePath)
			m.lines = strings.Split(string(content), "\n")
			m.filePath = filePath
			m.cursorX = 0
			m.cursorY = 0
			m.viewStart = 0
			m.focused = "editor"
		}
	}
	return m, nil
}

func updateEditor(m model, msg tea.KeyMsg) (model, tea.Cmd) {
	switch m.mode {
	case "normal":
		switch msg.String() {
		case "h":
			if m.cursorX > 0 {
				m.cursorX--
			}
		case "j":
			if m.cursorY < len(m.lines)-1 {
				m.cursorY++
				if m.cursorY >= m.viewStart+m.termHeight-2 {
					m.viewStart++
				}
			}
		case "k":
			if m.cursorY > 0 {
				m.cursorY--
				if m.cursorY < m.viewStart {
					m.viewStart--
				}
			}
		case "l":
			if m.cursorX < len(m.lines[m.cursorY]) {
				m.cursorX++
			}
		case "i":
			m.mode = "insert"
		case ":":
			m.mode = "command"
			m.commandBuffer = ":"
		}
	case "insert":
		switch msg.String() {
		case "esc":
			m.mode = "normal"
		default:
			if len(msg.String()) == 1 {
				char := msg.String()
				line := m.lines[m.cursorY]
				m.lines[m.cursorY] = line[:m.cursorX] + char + line[m.cursorX:]
				m.cursorX++
			}
		}
	case "command":
		switch msg.String() {
		case "enter":
			if m.commandBuffer == ":w" {
				os.WriteFile(m.filePath, []byte(strings.Join(m.lines, "\n")), 0644)
			} else if m.commandBuffer == ":q" {
				return m, tea.Quit
			}
			m.mode = "normal"
			m.commandBuffer = ""
		case "esc":
			m.mode = "normal"
			m.commandBuffer = ""
		default:
			m.commandBuffer += msg.String()
		}
	}
	return m, nil
}

func (m model) View() string {
	explorerView := renderExplorer(m)
	editorView := renderEditor(m)
	statusBar := renderStatusBar(m)

	explorerStyle := lipgloss.NewStyle().Width(m.explorerWidth).Height(m.termHeight - 1)
	editorStyle := lipgloss.NewStyle().Width(m.termWidth - m.explorerWidth - 1).Background(lipgloss.Color("gray"))
	separatorStyle := lipgloss.NewStyle().Width(1).Height(m.termHeight - 1).Background(lipgloss.Color("gray"))

	left := explorerStyle.Render(explorerView)
	right := editorStyle.Render(editorView)
	separator := separatorStyle.Render("|")

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

func renderExplorer(m model) string {
	var s strings.Builder
	for i, file := range m.files {
		name := file.Name()
		if file.IsDir() {
			name += "/"
		}
		if i == m.selectedIndex && m.focused == "explorer" {
			s.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("blue")).Render(name))
		} else {
			s.WriteString(name)
		}
		s.WriteString("\n")
	}
	return s.String()
}

func renderEditor(m model) string {
	var s strings.Builder
	viewHeight := m.termHeight - 2
	for i := m.viewStart; i < m.viewStart+viewHeight && i < len(m.lines); i++ {
		line := m.lines[i]
		if i == m.cursorY && m.focused == "editor" {
			if m.cursorX < len(line) {
				s.WriteString(line[:m.cursorX])
				s.WriteString(lipgloss.NewStyle().Reverse(true).Render(string(line[m.cursorX])))
				s.WriteString(line[m.cursorX+1:])
			} else {
				s.WriteString(line)
				s.WriteString(lipgloss.NewStyle().Reverse(true).Render(" "))
			}
		} else {
			s.WriteString(line)
		}
		s.WriteString("\n")
	}
	return s.String()
}

func renderStatusBar(m model) string {
	status := fmt.Sprintf("Mode: %s | File: %s", m.mode, m.filePath)
	if m.mode == "command" {
		status = m.commandBuffer
	}
	return lipgloss.NewStyle().Width(m.termWidth).Background(lipgloss.Color("green")).Render(status)
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

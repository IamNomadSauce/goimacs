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
	currentDir      string
	files           []os.DirEntry
	selectedIndex   int
	focused         string
	panes           []pane // Single pane for the editor
	activePaneIndex int
	prefixKey       string
	termWidth       int
	termHeight      int
	explorerWidth   int
}

type pane struct {
	editor  editor
	x, y    int
	width   int
	height  int
	focused bool
}

type editor struct {
	filePath      string
	lines         []string
	cursorX       int
	cursorY       int
	viewStart     int
	mode          string
	commandBuffer string
}

func initialModel() model {
	currentDir, _ := os.Getwd()
	files, _ := os.ReadDir(currentDir)
	return model{
		currentDir:    currentDir,
		files:         files,
		selectedIndex: 0,
		focused:       "explorer",
		termWidth:     80,
		termHeight:    24,
		explorerWidth: 20,
		pane:          pane{}, // Initialize with default empty pane
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
		m.pane.width = m.termWidth - m.explorerWidth - 1
		m.pane.height = m.termHeight - 1
		m.pane.x = 0
		m.pane.y = 0
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
			m.pane.editor = editor{
				filePath:  filePath,
				lines:     strings.Split(string(content), "\n"),
				cursorX:   0,
				cursorY:   0,
				viewStart: 0,
				mode:      "normal",
			}
			m.pane.focused = true
			m.focused = "editor"
		}
	}
	return m, nil
}

func updateEditor(m model, msg tea.KeyMsg) (model, tea.Cmd) {
	e := &m.pane.editor
	switch e.mode {
	case "normal":
		switch msg.String() {
		case "h":
			if e.cursorX > 0 {
				e.cursorX--
			}
		case "j":
			if e.cursorY < len(e.lines)-1 {
				e.cursorY++
				if e.cursorY >= e.viewStart+m.pane.height-2 {
					e.viewStart++
				}
			}
		case "k":
			if e.cursorY > 0 {
				e.cursorY--
				if e.cursorY < e.viewStart {
					e.viewStart--
				}
			}
		case "l":
			if e.cursorX < len(e.lines[e.cursorY]) {
				e.cursorX++
			}
		case "i":
			e.mode = "insert"
		case ":":
			e.mode = "command"
			e.commandBuffer = ":"
		}
	case "insert":
		switch msg.String() {
		case "esc":
			e.mode = "normal"
		default:
			if len(msg.String()) == 1 {
				char := msg.String()
				line := e.lines[e.cursorY]
				e.lines[e.cursorY] = line[:e.cursorX] + char + line[e.cursorX:]
				e.cursorX++
			}
		}
	case "command":
		switch msg.String() {
		case "enter":
			if e.commandBuffer == ":w" {
				os.WriteFile(e.filePath, []byte(strings.Join(e.lines, "\n")), 0644)
			} else if e.commandBuffer == ":q" {
				return m, tea.Quit
			}
			e.mode = "normal"
			e.commandBuffer = ""
		case "esc":
			e.mode = "normal"
			e.commandBuffer = ""
		default:
			e.commandBuffer += msg.String()
		}
	}
	return m, nil
}

func (m model) View() string {
	explorerView := renderExplorer(m)
	editorView := renderEditor(m.pane.editor, m.pane.height)
	statusBar := renderStatusBar(m)

	explorerStyle := lipgloss.NewStyle().Width(m.explorerWidth).Height(m.termHeight - 1)
	editorStyle := lipgloss.NewStyle().Width(m.termWidth - m.explorerWidth - 1).Height(m.termHeight - 1)
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

func renderEditor(e editor, height int) string {
	if len(e.lines) == 0 {
		return "No file open"
	}
	var s strings.Builder
	viewHeight := height - 2 // Adjust for status bar and borders
	for i := e.viewStart; i < e.viewStart+viewHeight && i < len(e.lines); i++ {
		line := e.lines[i]
		if i == e.cursorY {
			if e.cursorX < len(line) {
				s.WriteString(line[:e.cursorX])
				s.WriteString(lipgloss.NewStyle().Reverse(true).Render(string(line[e.cursorX])))
				s.WriteString(line[e.cursorX+1:])
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
	e := m.pane.editor
	status := fmt.Sprintf("Mode: %s | File: %s", e.mode, e.filePath)
	if e.mode == "command" {
		status = e.commandBuffer
	}
	if e.filePath == "" {
		status = "Mode: normal | No file open"
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

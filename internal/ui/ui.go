package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type UI struct {
	enabled bool

	bold  lipgloss.Style
	dim   lipgloss.Style
	ok    lipgloss.Style
	warn  lipgloss.Style
	error lipgloss.Style
	label lipgloss.Style
}

func New(out *os.File) UI {
	enabled := shouldStyle(out)
	r := lipgloss.NewRenderer(out)

	return UI{
		enabled: enabled,

		bold:  r.NewStyle().Bold(true),
		dim:   r.NewStyle().Faint(true),
		ok:    r.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		warn:  r.NewStyle().Foreground(lipgloss.Color("3")).Bold(true),
		error: r.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		label: r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#559db6", Dark: "#a3ddef"}).Bold(true),
	}
}

func shouldStyle(out *os.File) bool {
	if out == nil {
		return false
	}
	if !term.IsTerminal(int(out.Fd())) {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	return true
}

func (u UI) Bold(s string) string  { return u.render(u.bold, s) }
func (u UI) Dim(s string) string   { return u.render(u.dim, s) }
func (u UI) OK(s string) string    { return u.render(u.ok, s) }
func (u UI) Warn(s string) string  { return u.render(u.warn, s) }
func (u UI) Error(s string) string { return u.render(u.error, s) }
func (u UI) Label(s string) string { return u.render(u.label, s) }

func (u UI) render(style lipgloss.Style, s string) string {
	if !u.enabled {
		return s
	}
	return style.Render(s)
}

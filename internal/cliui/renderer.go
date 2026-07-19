package cliui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

type Renderer struct {
	out    io.Writer
	color  bool
	styles styles
}

type styles struct {
	brand    lipgloss.Style
	section  lipgloss.Style
	question lipgloss.Style
	success  lipgloss.Style
	warning  lipgloss.Style
	failure  lipgloss.Style
	progress lipgloss.Style
	label    lipgloss.Style
	command  lipgloss.Style
}

type Field struct {
	Label string
	Value string
}

func New(out io.Writer) Renderer {
	return newRenderer(out, colorEnabled(out, os.Environ()))
}

func NewForTest(out io.Writer, color bool) Renderer {
	return newRenderer(out, color)
}

func newRenderer(out io.Writer, color bool) Renderer {
	if out == nil {
		out = io.Discard
	}
	return Renderer{
		out:   out,
		color: color,
		styles: styles{
			brand:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightBlue),
			section:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue),
			question: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue),
			success:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green),
			warning:  lipgloss.NewStyle().Foreground(lipgloss.Yellow),
			failure:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red),
			progress: lipgloss.NewStyle().Foreground(lipgloss.Cyan),
			label:    lipgloss.NewStyle(),
			command:  lipgloss.NewStyle().Foreground(lipgloss.Cyan),
		},
	}
}

func (r Renderer) ColorEnabled() bool { return r.color }

func (r Renderer) Brand(name, tagline string) error {
	_, err := fmt.Fprintf(r.out, "%s %s\n", r.render(r.styles.brand, name+":"), tagline)
	return err
}

func (r Renderer) Section(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.section, text))
	return err
}

func (r Renderer) Question(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.question, "? "+text))
	return err
}

func (r Renderer) Prompt(text string) error {
	_, err := fmt.Fprint(r.out, r.render(r.styles.question, "› "+text))
	return err
}

func (r Renderer) Success(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.success, "✓ "+text))
	return err
}

func (r Renderer) Warning(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.warning, "! "+text))
	return err
}

func (r Renderer) Failure(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.failure, "✗ "+text))
	return err
}

func (r Renderer) Progress(text string) error {
	_, err := fmt.Fprintln(r.out, r.render(r.styles.progress, "→ "+text))
	return err
}

func (r Renderer) Text(text string) error {
	_, err := fmt.Fprintln(r.out, text)
	return err
}

func (r Renderer) Fields(fields ...Field) error {
	width := 0
	for _, field := range fields {
		if n := lipgloss.Width(field.Label); n > width {
			width = n
		}
	}
	for _, field := range fields {
		padding := strings.Repeat(" ", width-lipgloss.Width(field.Label)+2)
		if _, err := fmt.Fprintf(r.out, "  %s%s%s\n", r.render(r.styles.label, field.Label), padding, field.Value); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) Next(commands ...string) error {
	if _, err := fmt.Fprintln(r.out); err != nil {
		return err
	}
	if err := r.Section("Next"); err != nil {
		return err
	}
	for _, command := range commands {
		if _, err := fmt.Fprintf(r.out, "  %s\n", r.render(r.styles.command, command)); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) render(style lipgloss.Style, text string) string {
	if !r.color {
		return text
	}
	return style.Render(text)
}

func colorEnabled(out io.Writer, env []string) bool {
	values := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	if values["NO_COLOR"] != "" || strings.EqualFold(values["TERM"], "dumb") || values["CI"] != "" {
		return false
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

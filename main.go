package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dolmen-go/kittyimg"
	"golang.org/x/image/draw"
)

type Song struct {
	Title  string  `json:"title"`
	Rating float32 `json:"rating"`
}

type Album struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Rating  float32  `json:"rating"`
	Songs   []Song   `json:"songs"`
}

var songs []Song

const listHeight = 14

type styles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	pagination   lipgloss.Style
	help         lipgloss.Style
	quitText     lipgloss.Style
}

type listKeyMap struct {
	rateUp   key.Binding
	rateDown key.Binding
	export   key.Binding
}

func newItemKeyMap() listKeyMap {
	return listKeyMap{
		rateUp: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "rate up"),
		),
		rateDown: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "rate down"),
		),
		export: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "export ratings as JSON"),
		),
	}
}

func newStyles(darkBG bool) styles {
	var s styles
	s.title = lipgloss.NewStyle().MarginLeft(2)
	s.item = lipgloss.NewStyle().PaddingLeft(4)
	s.selectedItem = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	s.pagination = list.DefaultStyles(darkBG).PaginationStyle.PaddingLeft(4)
	s.help = list.DefaultStyles(darkBG).HelpStyle.PaddingLeft(4).PaddingBottom(1)
	s.quitText = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	return s
}

type item struct {
	song *Song
}

func (i item) FilterValue() string { return i.song.Title }

type itemDelegate struct {
	styles *styles
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s: %.2f", index+1, i.song.Title, i.song.Rating)

	fn := d.styles.item.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.selectedItem.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list     list.Model
	filename string
	album    Album
	choice   string
	styles   styles
	quitting bool
}

func initialModel(data []byte, filename string) model {
	album := Album{}
	if err := json.Unmarshal(data, &album); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	items := []list.Item{}
	for i := range album.Songs {
		items = append(items, item{song: &album.Songs[i]})
	}

	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)

	customKeys := newItemKeyMap()

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.rateUp, customKeys.rateDown, customKeys.export}
	}

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.rateUp, customKeys.rateDown, customKeys.export}
	}

	authors := strings.Join(album.Authors, ", ")

	l.Title = fmt.Sprintf("Rating '%s' by %s", album.Title, authors)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	m := model{list: l, album: album, filename: filename}
	m.updateStyles(true) // default to dark styles.
	return m
}

func (m *model) updateStyles(isDark bool) {
	m.styles = newStyles(isDark)
	m.list.Styles.Title = m.styles.title
	m.list.Styles.PaginationStyle = m.styles.pagination
	m.list.Styles.HelpStyle = m.styles.help
	m.list.SetDelegate(itemDelegate{styles: &m.styles})
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyPressMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			err := m.saveRatings()
			if err == nil {
				return m, m.list.NewStatusMessage("Successfully saved ratings!")
			}
			return m, nil
		case "right":
			idx := m.list.Index()
			i, ok := m.list.SelectedItem().(item)
			if ok {
				i.song.Rating += 0.5

				if i.song.Rating > 5 {
					i.song.Rating = 5
				}

				total := float32(0.0)
				for _, s := range m.album.Songs {
					total += s.Rating
				}
				m.album.Rating = total / float32(len(m.album.Songs))

				cmd := m.list.SetItem(idx, i)
				return m, cmd
			}
			return m, nil
		case "left":
			idx := m.list.Index()
			i, ok := m.list.SelectedItem().(item)
			if ok {
				i.song.Rating -= 0.5

				if i.song.Rating < 0 {
					i.song.Rating = 0
				}

				total := float32(0.0)
				for _, s := range m.album.Songs {
					total += s.Rating
				}
				m.album.Rating = total / float32(len(m.album.Songs))

				cmd := m.list.SetItem(idx, i)
				return m, cmd
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView(m.styles.quitText.Render("Bye"))
	}

	listView := m.list.View()

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		MarginTop(1).
		Bold(true)

	overallRating := fmt.Sprintf(" OVERALL ALBUM RATING: %.2f / 5.00 ", m.album.Rating)
	footer := footerStyle.Render(overallRating)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, listView, footer))
}

func (m model) saveRatings() error {
	bytes, err := json.MarshalIndent(m.album, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filename, bytes, 0644)
}

func main() {
	filename := "wlr.json"
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(initialModel(data, filename)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func printImageKitty() {
	file, _ := os.Open("wlr.jpg")
	defer file.Close()
	src, _, _ := image.Decode(file)

	width := 300
	height := (src.Bounds().Dy() * width) / src.Bounds().Dx()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	kittyimg.Fprintln(os.Stdout, dst)
}

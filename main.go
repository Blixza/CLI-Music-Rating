package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"rymcli/localizations"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

const listHeight = 20

type styles struct {
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	pagination   lipgloss.Style
	help         lipgloss.Style
	quitText     lipgloss.Style
}

type listKeyMap struct {
	cursorUp    key.Binding
	cursorDown  key.Binding
	nextPage    key.Binding
	prevPage    key.Binding
	goStart     key.Binding
	goEnd       key.Binding
	quit        key.Binding
	help        key.Binding
	closeHelp   key.Binding
	filter      key.Binding
	clearFilter key.Binding
	rateUp      key.Binding
	rateDown    key.Binding
	export      key.Binding
}

func newItemKeyMap(l *localizations.Localizer) listKeyMap {
	return listKeyMap{
		cursorUp: key.NewBinding(
			key.WithKeys("up", "j"),
			key.WithHelp("↑/j", l.Get("messages.cursor_up")),
		),
		cursorDown: key.NewBinding(
			key.WithKeys("down", "k"),
			key.WithHelp("↓/k", l.Get("messages.cursor_down")),
		),
		nextPage: key.NewBinding(
			key.WithKeys("l", "pgdn"),
			key.WithHelp("l/pgdn", l.Get("messages.next_page")),
		),
		prevPage: key.NewBinding(
			key.WithKeys("h", "pgup"),
			key.WithHelp("h/pgup", l.Get("messages.prev_page")),
		),
		goStart: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", l.Get("messages.go_to_start")),
		),
		goEnd: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", l.Get("messages.go_to_end")),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", l.Get("messages.quit")),
		),
		help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", l.Get("messages.show_help")),
		),
		closeHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", l.Get("messages.close_help")),
		),
		filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", l.Get("messages.filter")),
		),
		clearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", l.Get("messages.clear_filter")),
		),
		rateUp: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", l.Get("messages.rate_up")),
		),
		rateDown: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", l.Get("messages.rate_down")),
		),
		export: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", l.Get("messages.save_ratings")),
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
	list      list.Model
	filename  string
	album     Album
	choice    string
	styles    styles
	quitting  bool
	localizer *localizations.Localizer
	lang      string
}

func initialModel(data []byte, filename string, lang string) model {
	album := Album{}
	if err := json.Unmarshal(data, &album); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	localizer := localizations.New("en", "ru")
	localizer.Locale = lang

	items := []list.Item{}
	for i := range album.Songs {
		items = append(items, item{song: &album.Songs[i]})
	}

	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)

	customKeys := newItemKeyMap(localizer)

	l.KeyMap.CursorUp = customKeys.cursorUp
	l.KeyMap.CursorDown = customKeys.cursorDown
	l.KeyMap.ShowFullHelp = customKeys.help
	l.KeyMap.CloseFullHelp = customKeys.closeHelp
	l.KeyMap.Filter = customKeys.filter
	l.KeyMap.ClearFilter = customKeys.clearFilter
	l.KeyMap.GoToStart = customKeys.goStart
	l.KeyMap.GoToEnd = customKeys.goEnd
	l.KeyMap.NextPage = customKeys.nextPage
	l.KeyMap.PrevPage = customKeys.prevPage
	l.KeyMap.Quit = customKeys.quit

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.rateUp, customKeys.rateDown, customKeys.export}
	}

	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{customKeys.rateUp, customKeys.rateDown, customKeys.export}
	}

	authors := strings.Join(album.Authors, ", ")

	l.Title = localizer.Get("messages.rating_album_by", &localizations.Replacements{"title": album.Title, "authors": authors})
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	m := model{list: l, album: album, filename: filename, localizer: localizer, lang: lang}
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
				return m, m.list.NewStatusMessage(m.localizer.Get("messages.saved_ratings"))
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
	listView := m.list.View()

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		MarginTop(1).
		Bold(true)

	overallRating := m.localizer.Get("messages.overall_rating", &localizations.Replacements{"rating": fmt.Sprintf("%.2f", m.album.Rating)})
	footer := footerStyle.Render(overallRating)

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, listView, footer))
	v.AltScreen = true
	return v
}

func (m model) saveRatings() error {
	bytes, err := json.MarshalIndent(m.album, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filename, bytes, 0644)
}

func main() {
	lang := flag.String("lang", "en", "The localization language.")
	filename := flag.String("json", "", "Path to the JSON file.")

	flag.Parse()

	if *filename == "" {
		fmt.Println("You must provide a JSON file!")
		return
	}

	data, err := os.ReadFile(*filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(initialModel(data, *filename, *lang)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

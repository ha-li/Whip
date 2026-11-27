// Package palette is the set of colours the user can give things in Whip,
// such as note tags. Each colour has a stable name, which is what other
// packages store, and the hex values the frontend draws it with, so the
// values can be retuned without touching saved data.
package palette

// Color is one colour in the palette.
type Color struct {
	// Name identifies the colour, e.g. "blue". It is what callers store.
	Name string `json:"name"`
	// Label is the colour's name as shown to the user, e.g. "Blue".
	Label string `json:"label"`
	// Background is a light fill, as a CSS hex colour.
	Background string `json:"background"`
	// Text is the colour for text on Background, as a CSS hex colour. It
	// contrasts with Background at least 4.5:1 (WCAG AA).
	Text string `json:"text"`
}

// colors is the palette, in the order it is shown.
var colors = []Color{
	{Name: "gray", Label: "Gray", Background: "#e9e8e4", Text: "#45484d"},
	{Name: "brown", Label: "Brown", Background: "#efe3d6", Text: "#70462a"},
	{Name: "red", Label: "Red", Background: "#fbe0dd", Text: "#9c2519"},
	{Name: "orange", Label: "Orange", Background: "#fde5cf", Text: "#8c450b"},
	{Name: "yellow", Label: "Yellow", Background: "#faefc2", Text: "#6e5300"},
	{Name: "green", Label: "Green", Background: "#dcf1dc", Text: "#21622b"},
	{Name: "teal", Label: "Teal", Background: "#d5f0ec", Text: "#115a52"},
	{Name: "blue", Label: "Blue", Background: "#dbe8fb", Text: "#1b54a0"},
	{Name: "purple", Label: "Purple", Background: "#ebe1f8", Text: "#5f309b"},
	{Name: "pink", Label: "Pink", Background: "#fadff0", Text: "#99226b"},
}

// All returns the palette's colours in display order. The slice is a copy.
func All() []Color {
	out := make([]Color, len(colors))
	copy(out, colors)
	return out
}

// Lookup returns the colour with the given name, and whether there is one.
func Lookup(name string) (Color, bool) {
	for _, c := range colors {
		if c.Name == name {
			return c, true
		}
	}
	return Color{}, false
}

// Palette hands the palette to the frontend. It is bound to the frontend.
type Palette struct{}

// New returns a Palette.
func New() *Palette { return &Palette{} }

// Colors returns the palette's colours in display order.
func (*Palette) Colors() []Color { return All() }

package color

type Color string

const (
	BlackBold   Color = "\033[30;1m"
	RedBold     Color = "\033[31;1m"
	GreenBold   Color = "\033[32;1m"
	YellowBold  Color = "\033[33;1m"
	BlueBold    Color = "\033[34;1m"
	MagentaBold Color = "\033[35;1m"
	CyanBold    Color = "\033[36;1m"
	WhiteBold   Color = "\033[37;1m"
	Black       Color = "\033[30m"
	Red         Color = "\033[31m"
	Green       Color = "\033[32m"
	Yellow      Color = "\033[33m"
	Blue        Color = "\033[34m"
	Magenta     Color = "\033[35m"
	Cyan        Color = "\033[36m"
	White       Color = "\033[37m"
)

// ColorsList can be iterated over to easily pick a different color. This list
// is intentionally setup for how colors appear in my terminal, it's easy to
// pick your own!
//
// list := []Color{Red, Green, Blue}
// for i := 0; ; i++{
// 	col := list[i] % len(list)
// }
var ColorsList = []Color{
	CyanBold,
	GreenBold,
	MagentaBold,
	YellowBold,
	BlueBold,
}

// Add the surrounding strings needed to colorize TTY outputs
func (c Color) Add(in string) string {
	if c == "" {
		return in
	}
	return string(c) + in + "\033[0m"
}

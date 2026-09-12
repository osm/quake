package ui

type Event string

const (
	Toggle       Event = "toggle"
	Open         Event = "open"
	Close        Event = "close"
	Up           Event = "up"
	Down         Event = "down"
	Left         Event = "left"
	Right        Event = "right"
	Select       Event = "select"
	Delete       Event = "delete"
	Help         Event = "help"
	Back         Event = "back"
	Home         Event = "home"
	End          Event = "end"
	PageUp       Event = "pgup"
	PageDown     Event = "pgdn"
	BindStandard Event = "bindstd"
)

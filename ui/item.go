package ui

// Value, Disabled and DisabledReason may run repeatedly during rendering and navigation.
// Keep them read-only; session calls can re-enter their evaluation.
type Item struct {
	Key       string
	Focusable bool
	Separator bool
	Label     string
	Value     func(*Session) string
	Select    func(*Session) error
	Delete    func(*Session) error
	Adjust    func(*Session, int) error
	Submenu   *Page
	Help      *Page
	Disabled  func(*Session) bool

	// A nonempty reason disables the item; Select opens a dialog explaining why.
	DisabledReason func(*Session) string
	progress       func(*Session) float64
}

func (i Item) selectable(s *Session) bool {
	if i.Separator {
		return false
	}
	if i.DisabledReason != nil && i.DisabledReason(s) != "" {
		return true
	}
	return (i.Focusable || i.Help != nil || i.action()) &&
		(i.Disabled == nil || !i.Disabled(s))
}

func (i Item) disabled(s *Session) bool {
	return (i.Disabled != nil && i.Disabled(s)) ||
		(i.DisabledReason != nil && i.DisabledReason(s) != "")
}

func (i Item) action() bool {
	return i.Select != nil || i.Delete != nil || i.Adjust != nil || i.Submenu != nil
}

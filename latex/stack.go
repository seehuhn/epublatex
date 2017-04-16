package latex

// State contains information about the LaTeX rendering engine, which
// is reset at the end of a LaTeX group.
type State struct {
	isItalic bool
	isBold   bool
}

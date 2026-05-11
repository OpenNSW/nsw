package uiprojector

// Built-in projector keys. These are the names under which DefaultProjectors
// registers the projectors shipped with this package. Consumers may use any
// other string as a key when registering additional projectors.
const (
	ProjectorForm     = "FORM"
	ProjectorMarkdown = "MARKDOWN"
	ProjectorRaw      = "RAW"
)

// DefaultProjectors returns a fresh map containing the projectors shipped with
// this package. The returned map is owned by the caller and safe to mutate —
// add, override, or delete entries before passing it to NewAssembler.
func DefaultProjectors() map[string]Projector {
	return map[string]Projector{
		ProjectorForm:     NewFormProjector(),
		ProjectorMarkdown: NewMarkdownProjector(),
		ProjectorRaw:      NewRawProjector(),
	}
}

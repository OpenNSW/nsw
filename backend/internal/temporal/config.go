package temporal

// Config holds configuration required to connect to Temporal.
//
// This is owned by the temporal package (similar to other internal packages),
// so the package controls the shape/semantics of its configuration.
//
// Host/Port are kept separate to make configuration via environment variables
// easier and more explicit.
type Config struct {
	Host      string
	Port      int
	Namespace string
}

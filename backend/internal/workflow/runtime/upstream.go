package runtime

type UpstreamService interface {
	CompletionHandler(workflowID string, finalContext map[string]any) error
}

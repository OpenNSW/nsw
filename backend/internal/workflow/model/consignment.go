package model

// ConsignmentFlow represents the flow type of a consignment.
// Kept here (rather than imported from internal/consignment) so that
// workflow_template_map and template service signatures don't pull in
// the consignment package. Keep values in sync with consignment.Flow.
type ConsignmentFlow string

const (
	ConsignmentFlowImport ConsignmentFlow = "IMPORT"
	ConsignmentFlowExport ConsignmentFlow = "EXPORT"
)

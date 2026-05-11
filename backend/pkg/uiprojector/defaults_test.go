package uiprojector_test

import (
	"testing"

	"github.com/OpenNSW/nsw/pkg/uiprojector"
	"github.com/stretchr/testify/assert"
)

func TestProjectorConstants(t *testing.T) {
	assert.Equal(t, "FORM", uiprojector.ProjectorForm)
	assert.Equal(t, "MARKDOWN", uiprojector.ProjectorMarkdown)
	assert.Equal(t, "RAW", uiprojector.ProjectorRaw)
}

func TestDefaultProjectors_RegistersBuiltIns(t *testing.T) {
	p := uiprojector.DefaultProjectors()

	assert.Len(t, p, 3)
	assert.IsType(t, &uiprojector.FormProjector{}, p[uiprojector.ProjectorForm])
	assert.IsType(t, &uiprojector.MarkdownProjector{}, p[uiprojector.ProjectorMarkdown])
	assert.IsType(t, &uiprojector.RawProjector{}, p[uiprojector.ProjectorRaw])
}

func TestDefaultProjectors_ReturnsIndependentMaps(t *testing.T) {
	a := uiprojector.DefaultProjectors()
	b := uiprojector.DefaultProjectors()

	delete(a, uiprojector.ProjectorForm)
	a["CUSTOM"] = uiprojector.NewRawProjector()

	assert.Contains(t, b, uiprojector.ProjectorForm, "mutating one map must not affect another")
	assert.NotContains(t, b, "CUSTOM")
}

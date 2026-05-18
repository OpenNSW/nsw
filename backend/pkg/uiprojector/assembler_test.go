package uiprojector_test

import (
	"context"
	"errors"
	"testing"

	"github.com/OpenNSW/nsw/pkg/uiprojector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTemplateProvider struct {
	templates map[string][]byte
	err       error
}

func (s *stubTemplateProvider) GetTemplate(_ context.Context, id string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	t, ok := s.templates[id]
	if !ok {
		return nil, errors.New("template not found: " + id)
	}
	return t, nil
}

type stubProjector struct {
	lastTemplate []byte
	lastData     any
	out          any
	err          error
}

func (p *stubProjector) Project(_ context.Context, template []byte, data any) (any, error) {
	p.lastTemplate = template
	p.lastData = data
	if p.err != nil {
		return nil, p.err
	}
	return p.out, nil
}

func TestNewAssembler(t *testing.T) {
	t.Run("panics on nil TemplateProvider", func(t *testing.T) {
		assert.Panics(t, func() {
			uiprojector.NewAssembler(nil, nil)
		})
	})

	t.Run("deep copies the projectors map", func(t *testing.T) {
		pMap := map[string]uiprojector.Projector{"P1": &stubProjector{}}
		tp := &stubTemplateProvider{}
		asm := uiprojector.NewAssembler(tp, pMap)

		// Mutate original map
		delete(pMap, "P1")
		pMap["P2"] = &stubProjector{}

		// Assembler should be unaffected
		ctx := context.Background()
		tp.templates = map[string][]byte{"t": []byte("x")}
		bp := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
			"zone1": {ID: "s", TemplateID: "t", Projector: "P1"},
		}}

		_, err := asm.Assemble(ctx, bp, uiprojector.Facts{})
		assert.NoError(t, err, "Assembler should still have P1")

		bp2 := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
			"zone1": {ID: "s", TemplateID: "t", Projector: "P2"},
		}}
		_, err = asm.Assemble(ctx, bp2, uiprojector.Facts{})
		assert.Error(t, err, "Assembler should NOT have P2")
	})
}

func TestAssembler_Assemble_HappyPath(t *testing.T) {
	ctx := context.Background()
	tp := &stubTemplateProvider{templates: map[string][]byte{
		"tpl-a": []byte("A"),
		"tpl-b": []byte("B"),
	}}
	pA := &stubProjector{out: "rendered-A"}
	pB := &stubProjector{out: "rendered-B"}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{"PA": pA, "PB": pB})

	blueprint := &uiprojector.Blueprint{
		ID: "bp",
		Sections: map[string]uiprojector.SectionBlueprint{
			"main":    {ID: "s1", Title: "First", TemplateID: "tpl-a", Projector: "PA", DataKey: "alpha"},
			"sidebar": {ID: "s2", Title: "Second", TemplateID: "tpl-b", Projector: "PB"},
		},
	}
	facts := uiprojector.Facts{
		State: "IN_PROGRESS",
		Data: map[string]any{
			"alpha": map[string]any{"x": 1},
			"beta":  "value",
		},
	}

	sections, err := asm.Assemble(ctx, blueprint, facts)
	require.NoError(t, err)
	require.Len(t, sections, 2)

	assert.Equal(t, "s1", sections["main"].ID)
	assert.Equal(t, "First", sections["main"].Title)
	assert.Equal(t, uiprojector.SectionType("PA"), sections["main"].Type)
	assert.Equal(t, "rendered-A", sections["main"].Content)
	assert.Equal(t, map[string]any{"x": 1}, pA.lastData, "DataKey should pluck alpha")
	assert.Equal(t, []byte("A"), pA.lastTemplate)

	assert.Equal(t, "s2", sections["sidebar"].ID)
	assert.Equal(t, uiprojector.SectionType("PB"), sections["sidebar"].Type)
	assert.Equal(t, facts.Data, pB.lastData, "empty DataKey should pass full Data")
}

func TestAssembler_Assemble_SkipsHiddenSections(t *testing.T) {
	ctx := context.Background()
	tp := &stubTemplateProvider{templates: map[string][]byte{"t": []byte("x")}}
	p := &stubProjector{out: "ok"}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{"P": p})

	blueprint := &uiprojector.Blueprint{
		Sections: map[string]uiprojector.SectionBlueprint{
			"visible": {ID: "visible", TemplateID: "t", Projector: "P"},
			"hidden": {ID: "hidden", TemplateID: "t", Projector: "P", VisibleWhen: &uiprojector.VisibleWhen{
				States: []string{"NEVER"},
			}},
		},
	}

	sections, err := asm.Assemble(ctx, blueprint, uiprojector.Facts{State: "ANY"})
	require.NoError(t, err)
	require.Len(t, sections, 1)
	assert.Contains(t, sections, "visible")
	assert.NotContains(t, sections, "hidden")
}

func TestAssembler_Assemble_EmptyBlueprint(t *testing.T) {
	ctx := context.Background()
	asm := uiprojector.NewAssembler(&stubTemplateProvider{}, nil)

	sections, err := asm.Assemble(ctx, &uiprojector.Blueprint{}, uiprojector.Facts{})
	require.NoError(t, err)
	assert.Empty(t, sections)
}

func TestAssembler_Assemble_BlueprintIsNil(t *testing.T) {
	ctx := context.Background()
	asm := uiprojector.NewAssembler(&stubTemplateProvider{}, nil)

	_, err := asm.Assemble(ctx, nil, uiprojector.Facts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blueprint is nil")
}

func TestAssembler_Assemble_TemplateFetchError(t *testing.T) {
	ctx := context.Background()
	fetchErr := errors.New("boom")
	tp := &stubTemplateProvider{err: fetchErr}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{"P": &stubProjector{}})

	bp := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
		"main": {ID: "s", TemplateID: "missing", Projector: "P"},
	}}

	_, err := asm.Assemble(ctx, bp, uiprojector.Facts{})
	require.Error(t, err)
	assert.ErrorIs(t, err, fetchErr)
	assert.Contains(t, err.Error(), "missing", "error should mention the template ID")
}

func TestAssembler_Assemble_UnknownProjector(t *testing.T) {
	ctx := context.Background()
	tp := &stubTemplateProvider{templates: map[string][]byte{"t": []byte("x")}}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{})

	bp := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
		"main": {ID: "s", TemplateID: "t", Projector: "GHOST"},
	}}

	_, err := asm.Assemble(ctx, bp, uiprojector.Facts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown projector")
	assert.Contains(t, err.Error(), "GHOST")
}

func TestAssembler_Assemble_ProjectorError(t *testing.T) {
	ctx := context.Background()
	tp := &stubTemplateProvider{templates: map[string][]byte{"t": []byte("x")}}
	projErr := errors.New("render failed")
	p := &stubProjector{err: projErr}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{"P": p})

	bp := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
		"main": {ID: "section-7", TemplateID: "t", Projector: "P"},
	}}

	_, err := asm.Assemble(ctx, bp, uiprojector.Facts{})
	require.Error(t, err)
	assert.ErrorIs(t, err, projErr)
	assert.Contains(t, err.Error(), "section-7", "error should mention the failing section ID")
}

func TestAssembler_Assemble_DataKeyMissingPassesNil(t *testing.T) {
	ctx := context.Background()
	tp := &stubTemplateProvider{templates: map[string][]byte{"t": []byte("x")}}
	p := &stubProjector{out: "ok"}
	asm := uiprojector.NewAssembler(tp, map[string]uiprojector.Projector{"P": p})

	bp := &uiprojector.Blueprint{Sections: map[string]uiprojector.SectionBlueprint{
		"main": {ID: "s", TemplateID: "t", Projector: "P", DataKey: "absent"},
	}}
	facts := uiprojector.Facts{Data: map[string]any{"other": 1}}

	_, err := asm.Assemble(ctx, bp, facts)
	require.NoError(t, err)
	assert.Nil(t, p.lastData, "DataKey lookup on missing key should pass nil")
}

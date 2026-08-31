package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/render"
)

var _ render.Renderer = (*Renderer)(nil)

// Renderer draws every Position+Appearance entity, running each
// AppearanceModifier in order to resolve its final draw layers.
type Renderer struct {
	renderQuery *goke.Query
	pos         goke.Comp[Position]
	appearance  goke.Comp[Appearance]
	modifiers   []AppearanceModifier
	layers      []Appearance
	batch       spriteBatch
}

func newRenderer(atlas render.AtlasSource) *Renderer {
	return &Renderer{batch: newSpriteBatch(atlas)}
}

// BindCamera attaches camera — Draw needs it, so call this before the first Draw.
func (s *Renderer) BindCamera(camera render.Camera) { s.batch.camera = camera }

func (s *Renderer) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.pos, &s.appearance)
	for _, m := range s.modifiers {
		m.Bind(qb)
	}
	s.renderQuery = qb.Build()
}

func (s *Renderer) Draw(screen *ebiten.Image) {
	s.batch.reset()

	s.renderQuery.All()
	for s.renderQuery.Next() {
		cursor := s.renderQuery.Cursor()
		positions := s.pos.Slice(cursor)
		appearances := s.appearance.Slice(cursor)

		for i := range cursor.IDs {
			s.layers = append(s.layers[:0], appearances[i])
			for _, m := range s.modifiers {
				s.layers = m.Apply(cursor, i, s.layers)
			}
			for _, l := range s.layers {
				s.batch.drawQuad(positions[i], l.SpriteID, l.Color)
			}
		}
	}

	s.batch.flush(screen)
}

// WithReplace adds a modifier replacing dst[0] with with, for every entity carrying T.
func (s *Renderer) WithReplace[T any](with Appearance) *Renderer {
	return s.WithStrategy(replace[T](with))
}

// WithOverlay adds a modifier appending with on top, for every entity carrying T.
func (s *Renderer) WithOverlay[T any](with Appearance) *Renderer {
	return s.WithStrategy(overlay[T](with))
}

// WithModify adds a modifier transforming dst[0] via f, for every entity carrying T.
func (s *Renderer) WithModify[T any](f func(Appearance, T) Appearance) *Renderer {
	return s.WithStrategy(modify[T](f))
}

// WithStrategy adds a modifier running strategy for every entity carrying T — the escape hatch for a custom AppearanceStrategy[T].
func (s *Renderer) WithStrategy[T any](strategy AppearanceStrategy[T]) *Renderer {
	return s.WithModifier(&conditionalApperanceStrategy[T]{Strategy: strategy})
}

// WithModifier appends m to the modifiers run, in order, for every entity.
func (s *Renderer) WithModifier(m AppearanceModifier) *Renderer {
	s.modifiers = append(s.modifiers, m)
	return s
}

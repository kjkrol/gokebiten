package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
)

var _ Renderer = (*EntitiesRenderer)(nil)

// EntitiesRenderer draws every Position+Appearance entity, running each
// Plugin in order to resolve its final draw layers.
type EntitiesRenderer struct {
	renderQuery *goke.Query
	pos         goke.Comp[kinematics.Position]
	appearance  goke.Comp[Appearance]
	plugins     []Plugin
	layers      []Appearance
	batch       spriteBatch
}

// NewEntitiesRenderer draws Position+Appearance entities — chain WithPlugin to add refinements.
func NewEntitiesRenderer(atlas AtlasSource, camera Camera) *EntitiesRenderer {
	return &EntitiesRenderer{batch: newSpriteBatch(atlas, camera)}
}

// WithPlugin appends p to the plugins run, in order, for every entity.
func (s *EntitiesRenderer) WithPlugin(p Plugin) *EntitiesRenderer {
	s.plugins = append(s.plugins, p)
	return s
}

// WithReplace adds a Component[T] plugin replacing dst[0] with with, for every entity carrying T.
func (s *EntitiesRenderer) WithReplace[T any](with Appearance) *EntitiesRenderer {
	return s.WithPlugin(&Component[T]{Strategy: Replace[T](with)})
}

// WithOverlay adds a Component[T] plugin appending with on top, for every entity carrying T.
func (s *EntitiesRenderer) WithOverlay[T any](with Appearance) *EntitiesRenderer {
	return s.WithPlugin(&Component[T]{Strategy: Overlay[T](with)})
}

// WithModify adds a Component[T] plugin transforming dst[0] via f, for every entity carrying T.
func (s *EntitiesRenderer) WithModify[T any](f func(Appearance, T) Appearance) *EntitiesRenderer {
	return s.WithPlugin(&Component[T]{Strategy: Modify[T](f)})
}

func (s *EntitiesRenderer) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.pos, &s.appearance)
	for _, p := range s.plugins {
		p.Bind(qb)
	}
	s.renderQuery = qb.Build()
}

func (s *EntitiesRenderer) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 50, G: 50, B: 50, A: 255})
	s.batch.reset()

	s.renderQuery.All()
	for s.renderQuery.Next() {
		cursor := s.renderQuery.Cursor()
		positions := s.pos.Slice(cursor)
		appearances := s.appearance.Slice(cursor)

		for i := range cursor.IDs {
			s.layers = append(s.layers[:0], appearances[i])
			for _, p := range s.plugins {
				s.layers = p.Apply(cursor, i, s.layers)
			}
			for _, l := range s.layers {
				s.batch.drawQuad(positions[i], l.SpriteID, l.Color)
			}
		}
	}

	s.batch.flush(screen)
}

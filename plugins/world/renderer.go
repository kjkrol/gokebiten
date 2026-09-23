package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
)

var _ render.Renderer = (*Renderer)(nil)

// Renderer draws the Position+Appearance entities the Space finds in the camera's bounds, running
// each AppearanceModifier in order to resolve their final draw layers; with the whole world in view
// it walks every entity instead. An entity not yet in the Space (spawned outside Populate) is drawn
// from the next Rebuild on.
type Renderer struct {
	renderQuery *goke.Query
	base        goke.Comp[Base]
	appearance  goke.Comp[Appearance]
	modifiers   []AppearanceModifier
	layers      []Appearance
	batch       spriteBatch

	space     *aabbworld.Space
	camera    camera.Camera
	worldArea float64
	// hit is a bit per entity index, set for what the Space found in view this frame.
	hit  []uint64
	mark func(uid.UID64)
}

func newRenderer(cam camera.Camera, atlas render.AtlasSource, space *aabbworld.Space, worldW, worldH uint32) *Renderer {
	r := &Renderer{
		batch:     newSpriteBatch(cam, atlas, worldW, worldH),
		space:     space,
		camera:    cam,
		worldArea: float64(worldW) * float64(worldH),
	}
	r.mark = r.markHit
	return r
}

func (s *Renderer) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.base, &s.appearance)
	for _, m := range s.modifiers {
		m.Bind(qb)
	}
	s.renderQuery = qb.Build()
}

// Draw draws this frame; a nil screen gathers the quads and draws nothing, for measuring.
func (s *Renderer) Draw(screen *ebiten.Image) {
	s.batch.reset()
	culled := s.cull()

	s.renderQuery.All()
	for s.renderQuery.Next() {
		cursor := s.renderQuery.Cursor()
		bases := s.base.Slice(cursor)
		appearances := s.appearance.Slice(cursor)

		for i, id := range cursor.IDs {
			if culled && !s.inView(id.Index()) {
				continue
			}
			s.layers = append(s.layers[:0], appearances[i])
			for _, m := range s.modifiers {
				s.layers = m.Apply(cursor, i, &bases[i], s.layers)
			}
			for _, l := range s.layers {
				s.batch.drawQuad(bases[i].Pos, l.SpriteID)
			}
		}
	}

	s.batch.flush(screen)
}

// cull marks every entity the Space finds in the camera's bounds; false when the bounds cover
// the whole world, in which case nothing is marked and everything is drawn.
func (s *Renderer) cull() bool {
	b := s.camera.Bounds()
	if (b.BottomRight.X-b.TopLeft.X)*(b.BottomRight.Y-b.TopLeft.Y) >= s.worldArea {
		return false
	}
	clear(s.hit)
	s.space.Query(b, aabbworld.AnyCapability, s.mark)
	return true
}

func (s *Renderer) markHit(id uid.UID64) {
	i := id.Index()
	w := int(i >> 6)
	for w >= len(s.hit) {
		s.hit = append(s.hit, 0)
	}
	s.hit[w] |= 1 << (i & 63)
}

func (s *Renderer) inView(i uint32) bool {
	w := int(i >> 6)
	return w < len(s.hit) && s.hit[w]&(1<<(i&63)) != 0
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

// WithStrategy adds a modifier running strategy for every entity carrying T.
func (s *Renderer) WithStrategy[T any](strategy AppearanceStrategy[T]) *Renderer {
	return s.WithModifier(&conditionalApperanceStrategy[T]{Strategy: strategy})
}

// WithModifier appends m to the modifiers run, in order, for every entity.
func (s *Renderer) WithModifier(m AppearanceModifier) *Renderer {
	s.modifiers = append(s.modifiers, m)
	return s
}

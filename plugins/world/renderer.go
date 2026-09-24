package world

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
)

var _ render.Renderer = (*Renderer)(nil)

// Renderer draws the Position+Appearance entities in the world's View — what the camera sees
// this tick — running the Each behaviors of a Drawing over each chunk to settle their layers. A
// Stage that has not ticked yet sees everything.
type Renderer struct {
	renderQuery *goke.Query
	base        goke.Comp[Base]
	appearance  goke.Comp[Appearance]
	host        *plugin.EachHost[Drawing]
	layers      [][]Appearance // one per entity of the chunk being drawn
	batch       spriteBatch
	view        *View

	ids   []uid.UID64
	bases []Base
}

func newRenderer(cam camera.Camera, atlas render.AtlasSource, view *View, host *plugin.EachHost[Drawing], worldW, worldH uint32) *Renderer {
	return &Renderer{batch: newSpriteBatch(cam, atlas, worldW, worldH), view: view, host: host}
}

func (s *Renderer) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&s.base, &s.appearance)
	s.host.Bind(qb)
	s.renderQuery = qb.Build()
}

// Draw draws this frame; a nil screen gathers the quads and draws nothing, for measuring.
func (s *Renderer) Draw(screen *ebiten.Image) {
	s.batch.reset()
	tick := plugin.Tick{Now: time.Now()}

	s.renderQuery.All()
	for s.renderQuery.Next() {
		cursor := s.renderQuery.Cursor()
		s.ids, s.bases = cursor.IDs, s.base.Slice(cursor)
		appearances := s.appearance.Slice(cursor)

		for len(s.layers) < len(s.ids) {
			s.layers = append(s.layers, nil)
		}
		for i := range s.ids {
			s.layers[i] = append(s.layers[i][:0], appearances[i])
		}
		if !s.host.Empty() {
			s.host.Run(tick, cursor, s.at)
		}
		for i, id := range s.ids {
			if !s.view.Contains(id) {
				continue
			}
			for _, l := range s.layers[i] {
				s.batch.drawQuad(s.bases[i].Pos, l.SpriteID)
			}
		}
	}

	s.batch.flush(screen)
}

// at describes the i-th entity of the chunk being drawn.
func (s *Renderer) at(i int) Drawing {
	return Drawing{ID: s.ids[i], Base: &s.bases[i], Layers: &s.layers[i]}
}

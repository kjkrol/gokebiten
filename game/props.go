package game

// Props configures Engine's window and target tick rate — returned by
// Game.Props and read once, when the engine starts.
type Props struct {
	Title                     string
	TargetTPS                 int
	ScreenWidth, ScreenHeight int
}

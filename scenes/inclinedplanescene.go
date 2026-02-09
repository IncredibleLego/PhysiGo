package scenes

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type InclinedPlaneScene struct {
}

func (i *InclinedPlaneScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return reason == Unpause
}

func NewInclinedPlaneScene() *InclinedPlaneScene {
	return &InclinedPlaneScene{}
}

func (i *InclinedPlaneScene) Draw(screen *ebiten.Image) {
	// Black screen - nothing to draw
}

func (i *InclinedPlaneScene) FirstLoad() {
	// Nothing to initialize yet
}

func (i *InclinedPlaneScene) OnEnter() {
}

func (i *InclinedPlaneScene) OnExit() {
}

func (i *InclinedPlaneScene) Update() SceneId {
	// Open options menu with Enter key
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return PauseSceneId
	}

	return InclinedPlaneSceneId
}

var _ Scene = (*InclinedPlaneScene)(nil)

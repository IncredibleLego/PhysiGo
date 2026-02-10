package scenes

import (
	"physiGo/config"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type InclinedPlaneScene struct {
	theta   float64
	muS     float64
	muK     float64
	mass    float64
	gravity float64
	length  float64
	hBlock  float64

	muSSet     bool
	muKSet     bool
	gravitySet bool
}

func (i *InclinedPlaneScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return reason == Unpause
}

func NewInclinedPlaneScene() *InclinedPlaneScene {
	return &InclinedPlaneScene{}
}

func (i *InclinedPlaneScene) Draw(screen *ebiten.Image) {
	screen.Clear()
}

func (i *InclinedPlaneScene) FirstLoad() {
	i.theta = config.GlobalConfig.InclinedTheta
	i.muS = config.GlobalConfig.InclinedMuS
	i.muK = config.GlobalConfig.InclinedMuK
	i.mass = config.GlobalConfig.InclinedMass
	i.gravity = config.GlobalConfig.InclinedGravity
	i.length = config.GlobalConfig.InclinedLength
	i.hBlock = config.GlobalConfig.InclinedHBlock
	i.muSSet = config.GlobalConfig.InclinedMuSSet
	i.muKSet = config.GlobalConfig.InclinedMuKSet
	i.gravitySet = config.GlobalConfig.InclinedGravitySet
}

func (i *InclinedPlaneScene) OnEnter() {
}

func (i *InclinedPlaneScene) OnExit() {
}

func (i *InclinedPlaneScene) Update() SceneId {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return PauseSceneId
	}

	return InclinedPlaneSceneId
}

var _ Scene = (*InclinedPlaneScene)(nil)

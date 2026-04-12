package scenes

import (
	"math"
	"physiGo/config"
	"physiGo/utils"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type CreditsScene struct{}

func NewCreditsScene() *CreditsScene {
	return &CreditsScene{}
}

func (c *CreditsScene) FirstLoad() {}

func (c *CreditsScene) OnEnter() {}

func (c *CreditsScene) OnExit() {}

func (c *CreditsScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return true
}

func (c *CreditsScene) Update() SceneId {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return StartSceneId
	}
	return CreditsSceneId
}

func fitFontSizeForText(text string, baseSize float64, fontName string, maxWidth float64) float64 {
	if baseSize <= 1 {
		return 1
	}

	w, _ := utils.MeasureTextWithSize(text, baseSize, fontName)
	if w <= maxWidth {
		return baseSize
	}

	ratio := maxWidth / w
	fitted := math.Floor(baseSize * ratio)
	if fitted < 1 {
		return 1
	}
	return fitted
}

func drawCenteredLine(screen *ebiten.Image, y float64, colorName, lineText, fontName string, size float64) {
	x := utils.XCenteredWithFont(lineText, size, fontName)
	utils.ScreenDraw(size-config.GlobalConfig.TextDimension, x, y, colorName, screen, lineText, fontName)
}

func (c *CreditsScene) Draw(screen *ebiten.Image) {
	y := float64(config.GlobalConfig.ScreenHeight) * 0.15
	baseLine := config.GlobalConfig.TextDimension * 1.7
	fontSize := config.GlobalConfig.TextDimension
	fontName := "pressStart"
	maxTextWidth := float64(config.GlobalConfig.ScreenWidth) * 0.92

	drawCenteredLine(screen, y, "white", "Made by", fontName, fontSize)
	y += baseLine
	drawCenteredLine(screen, y, "sky blue", "Francesco Corrado", fontName, fontSize)

	y += baseLine * 1.35
	drawCenteredLine(screen, y, "white", "Fonts Used", fontName, fontSize)
	y += baseLine
	fontsLine := "LibertinusMath, PressStart2P"
	fontsSize := fitFontSizeForText(fontsLine, fontSize, fontName, maxTextWidth)
	drawCenteredLine(screen, y, "sky blue", fontsLine, fontName, fontsSize)

	y += baseLine * 1.35
	drawCenteredLine(screen, y, "white", "Thanks to", fontName, fontSize)

	thanksLines := []string{
		"Mario Corrado, Luca Ghirimoldi",
		"Federico Gallo, Federico Falcone",
		"Riccardo Cavadini",
	}
	for _, lineText := range thanksLines {
		y += baseLine
		nameSize := fitFontSizeForText(lineText, fontSize, fontName, maxTextWidth)
		drawCenteredLine(screen, y, "sky blue", lineText, fontName, nameSize)
	}

	footerSize := fontSize - (config.GlobalConfig.TextDimension / 2.6)
	if footerSize < 1 {
		footerSize = 1
	}
	drawCenteredLine(screen, float64(config.GlobalConfig.ScreenHeight)*0.92, "light gray", "Press ENTER/Mouse to go back", fontName, footerSize)
}

var _ Scene = (*CreditsScene)(nil)

package scenes

import (
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"physiGo/config"
	"physiGo/utils"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type SceneId uint

const (
	GameSceneId SceneId = iota
	StartSceneId
	ExitSceneId
	PauseSceneId
	OptionsSceneId
	InclinedInputSceneId
	InclinedPlaneSceneId
	ProjectileMotionInputSceneId
	ProjectileMotionSceneId
	CreditsSceneId
)

type Scene interface {
	Update() SceneId
	Draw(screen *ebiten.Image)
	FirstLoad()
	OnEnter()
	OnExit()
	ShouldPreserveState(reason SceneChangeReason) bool
}

type SceneChangeReason string

const (
	Unpause SceneChangeReason = "unpause"
	Exit    SceneChangeReason = "exit"
	Other   SceneChangeReason = "other"
)

type uiRect struct {
	x float64
	y float64
	w float64
	h float64
}

// contains verifica se un punto e` dentro il rettangolo UI.
func (r uiRect) contains(px, py float64) bool {
	return px >= r.x && px <= r.x+r.w && py >= r.y && py <= r.y+r.h
}

// clamp01 limita un valore al range [0,1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// timelineRects calcola i rettangoli di bottone play/pause e barra timeline.
func timelineRects() (uiRect, uiRect) {
	textDim := config.GlobalConfig.TextDimension
	barX := float64(config.GlobalConfig.ScreenWidth) * 0.2
	barY := float64(config.GlobalConfig.ScreenHeight) - textDim*2.0
	barW := float64(config.GlobalConfig.ScreenWidth) * 0.6
	barH := textDim * 0.45

	buttonSize := textDim * 1.25
	buttonX := barX - buttonSize - textDim*0.45
	buttonY := barY - (buttonSize-textDim*0.5)/2

	return uiRect{x: buttonX, y: buttonY, w: buttonSize, h: buttonSize}, uiRect{x: barX, y: barY, w: barW, h: barH}
}

// drawTimelineButton disegna il bottone play/pause con icona o testo fallback.
func drawTimelineButton(screen *ebiten.Image, rect uiRect, running bool, playImage, pauseImage *ebiten.Image, fallbackLabel string) {
	vector.DrawFilledRect(screen, float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h), color.RGBA{40, 40, 40, 255}, false)

	icon := playImage
	if running {
		icon = pauseImage
	}

	if icon != nil {
		imgW, imgH := icon.Bounds().Dx(), icon.Bounds().Dy()
		if imgW > 0 && imgH > 0 {
			scale := math.Min(rect.w*0.8/float64(imgW), rect.h*0.8/float64(imgH))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scale, scale)
			drawW := float64(imgW) * scale
			drawH := float64(imgH) * scale
			op.GeoM.Translate(rect.x+(rect.w-drawW)/2, rect.y+(rect.h-drawH)/2)
			screen.DrawImage(icon, op)
			return
		}
	}

	if fallbackLabel == "" {
		return
	}
	label := fallbackLabel
	if running {
		label = "PAUSE"
	}
	x := rect.x + rect.w*0.12
	y := rect.y + rect.h*0.68
	utils.ScreenDraw(-(config.GlobalConfig.TextDimension / 2.2), x, y, "white", screen, label, "libertinus")
}

// drawTimelineBar disegna barra progresso, marker opzionale e cursore corrente.
func drawTimelineBar(screen *ebiten.Image, rect uiRect, progress, baseProgress float64) {
	progress = clamp01(progress)
	vector.DrawFilledRect(screen, float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h), color.RGBA{55, 55, 55, 255}, false)
	vector.DrawFilledRect(screen, float32(rect.x), float32(rect.y), float32(rect.w*progress), float32(rect.h), color.RGBA{30, 170, 90, 255}, false)

	if baseProgress >= 0 {
		baseX := rect.x + rect.w*clamp01(baseProgress)
		vector.DrawFilledRect(screen, float32(baseX-2), float32(rect.y-3), 4, float32(rect.h+6), color.RGBA{240, 95, 40, 255}, false)
	}

	knobX := rect.x + rect.w*progress
	vector.DrawFilledRect(screen, float32(knobX-2), float32(rect.y-4), 4, float32(rect.h+8), color.RGBA{230, 230, 230, 255}, false)
}

// timelineScrubDelta restituisce il delta tempo di scrub con frecce, se attivo.
func timelineScrubDelta(total float64) (float64, bool) {
	if total <= 0 {
		return 0, false
	}

	step := total * 0.02
	if step < 0.05 {
		step = 0.05
	}

	if keyPressedOrHeld(ebiten.KeyArrowRight) {
		return step, true
	}
	if keyPressedOrHeld(ebiten.KeyArrowLeft) {
		return -step, true
	}
	return 0, false
}

// keyPressedOrHeld ritorna true su pressione singola o ripetizione da pressione lunga.
func keyPressedOrHeld(key ebiten.Key) bool {
	if inpututil.IsKeyJustPressed(key) {
		return true
	}
	dur := inpututil.KeyPressDuration(key)
	return dur > 10 && dur%3 == 0
}

// progressFromCursorX converte la posizione mouse in progresso normalizzato [0,1].
func progressFromCursorX(px float64, bar uiRect) float64 {
	if bar.w <= 0 {
		return 0
	}
	return clamp01((px - bar.x) / bar.w)
}

// loadImage carica un'immagine da disco e la converte in ebiten.Image.
func loadImage(path string) *ebiten.Image {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// renderBlinkingValue mostra il cursore lampeggiante per il campo input attivo.
func renderBlinkingValue(lastBlink *time.Time, value string) string {
	blinkOn := time.Since(*lastBlink) < time.Second
	if time.Since(*lastBlink) > 2*time.Second {
		*lastBlink = time.Now()
	}
	if blinkOn {
		return value + "_"
	}
	if value == "" {
		return "-"
	}
	return value
}

// handleNumericTextInput gestisce input numerico base: cifre, backspace e separatore decimale.
// Se overwriteOnType e' true, il primo carattere numerico sostituisce completamente il valore corrente.
func handleNumericTextInput(input *string, maxChars int, overwriteOnType *bool) {
	text := *input
	overwrite := overwriteOnType != nil && *overwriteOnType

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(text) > 0 {
		text = text[:len(text)-1]
	}

	for key := ebiten.Key0; key <= ebiten.Key9; key++ {
		if inpututil.IsKeyJustPressed(key) {
			if overwrite {
				text = ""
				overwrite = false
			}
			if len(text) < maxChars {
				text += string('0' + rune(key-ebiten.Key0))
			}
			if overwriteOnType != nil {
				*overwriteOnType = false
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) || inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		if overwrite {
			text = ""
			overwrite = false
		}
		if !strings.ContainsAny(text, ".,") && len(text) < maxChars {
			if text == "" {
				text = "0"
			}
			text += "."
		}
		if overwriteOnType != nil {
			*overwriteOnType = false
		}
	}

	*input = text
}

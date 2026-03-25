package scenes

import (
	"fmt"
	"math"
	"physiGo/config"
	"physiGo/menu"
	"physiGo/utils"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenOptionsMenuName = "OPTIONS"
)

type OptionScene struct {
	screenMenu      *menu.OptionMenu
	scalePopup      *utils.Popup
	previousSceneId SceneId
	savedScale      float64
	lastAdjustTime  time.Time
}

func NewOptionScene(previous SceneId) *OptionScene {
	return &OptionScene{
		screenMenu:      nil,
		previousSceneId: previous,
	}
}

func (o *OptionScene) generateScreenMenuOptions() []string {
	// Calculate the screen size based on the current scale (even if not yet applied)
	scale := config.GlobalConfig.Scale
	width := ((int(math.Round(float64(config.DefaultConfig.ScreenWidth)*scale)) + 5) / 10) * 10
	height := ((int(math.Round(float64(config.DefaultConfig.ScreenHeight)*scale)) + 5) / 10) * 10

	return []string{
		"Text Dimension: " + strconv.Itoa(int(config.GlobalConfig.TextDimension)),
		"Screen Size: " + strconv.Itoa(width) + " x " + strconv.Itoa(height),
		"FullScreen: " + strconv.FormatBool(config.GlobalConfig.Fullscreen),
		"Reset to default",
		"Back",
	}
}
func (o *OptionScene) Draw(screen *ebiten.Image) {
	o.screenMenu.Draw(screen)

	if o.scalePopup.Active {
		o.scalePopup.Draw(screen)
	}
}

func (o *OptionScene) FirstLoad() {
	o.screenMenu = &menu.OptionMenu{
		Options:      o.generateScreenMenuOptions(),
		Selected:     0,
		LastMoveTime: time.Now(),
		MenuName:     screenOptionsMenuName,
		Position:     (float64(config.GlobalConfig.ScreenHeight) * 0.3125),
	}
	o.scalePopup = &utils.Popup{
		Active:  false,
		Text:    "Scale has changed, restart the game to apply changes",
		Options: []string{"YES", "NO"},
	}
	o.savedScale = config.GlobalConfig.Scale
	o.lastAdjustTime = time.Now()
}

func (o *OptionScene) OnEnter() {}

func (o *OptionScene) OnExit() {}

func (o *OptionScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return false
}

func (o *OptionScene) Update() SceneId {
	o.screenMenu.Options = o.generateScreenMenuOptions()
	o.screenMenu.Update()

	if o.scalePopup.Active {
		o.scalePopup.Update()
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return o.handleExitPopup()
		}
	}

	if time.Since(o.lastAdjustTime) >= config.GlobalConfig.OptionsPerSecond {
		if inpututil.KeyPressDuration(ebiten.KeyArrowRight) > 0 || inpututil.KeyPressDuration(ebiten.KeyD) > 0 {
			handleScreenMenuOptions(o, o.screenMenu.Selected, true)
			o.lastAdjustTime = time.Now()
		}
		if inpututil.KeyPressDuration(ebiten.KeyArrowLeft) > 0 || inpututil.KeyPressDuration(ebiten.KeyA) > 0 {
			handleScreenMenuOptions(o, o.screenMenu.Selected, false)
			o.lastAdjustTime = time.Now()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && o.screenMenu.Selected == len(o.screenMenu.Options)-1 {
		if o.savedScale != config.GlobalConfig.Scale {
			o.scalePopup.Active = true
			o.scalePopup.Selected = 0
			return OptionsSceneId
		}
		return o.previousSceneId
	}

	return OptionsSceneId
}

func updateConfigValueFloat(option *float64, min, max, step float64, mode bool) bool {
	originalValue := *option
	err := config.UpdateConfig(func(cfg *config.Config) {
		if mode && *option < max {
			*option += step
		} else if !mode && *option > min {
			*option -= step
		}
	})
	if err != nil {
		fmt.Println("Error during option saving", err)
	}
	return originalValue != *option
}
func handleScreenMenuOptions(o *OptionScene, selectedOption int, mode bool) {
	// If mode is true = +, if false = -
	switch selectedOption {
	case 0:
		updateConfigValueFloat(&config.GlobalConfig.TextDimension, 1, 35, 1, mode)
	case 1:
		updateConfigValueFloat(&config.GlobalConfig.Scale, 0.67, 1.99, 0.33, mode)
	case 2:
		err := config.UpdateConfig(func(cfg *config.Config) {
			cfg.Fullscreen = !cfg.Fullscreen
		})
		if err != nil {
			fmt.Println("Error during option saving", err)
		}
	case 3:
		err := config.UpdateConfig(func(cfg *config.Config) {
			cfg.TextDimension = math.Round(config.DefaultConfig.TextDimension * config.GlobalConfig.Scale)
			cfg.Scale = config.DefaultConfig.Scale
			cfg.Fullscreen = config.DefaultConfig.Fullscreen

		})
		if err != nil {
			fmt.Println("Error during option saving", err)
		}
	}
}

func (o *OptionScene) handleExitPopup() SceneId {
	if o.scalePopup.Selected == 0 {
		err := config.ChangeScale(config.GlobalConfig.Scale)
		if err != nil {
			fmt.Println("Error during option saving", err)
		}
		utils.RestartGame()
	} else {
		o.scalePopup.Active = false
		config.GlobalConfig.Scale = o.savedScale
	}
	return OptionsSceneId
}

var _ Scene = (*OptionScene)(nil)

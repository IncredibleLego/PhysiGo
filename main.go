package main

import (
	"log"
	"physiGo/audio"
	"physiGo/config"

	"github.com/hajimehoshi/ebiten/v2"
)

var isFullscreen bool

func main() {
	config.InitConfig()
	config.ApplyScaleToConfig(config.GlobalConfig, config.GlobalConfig.Scale)
	audio.Init()
	//fmt.Printf("Loaded configuration: %+v\n", config.GlobalConfig)

	ebiten.SetWindowTitle("PhysiGo")
	ebiten.SetWindowSize(config.GlobalConfig.ScreenWidth, config.GlobalConfig.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	isFullscreen = config.GlobalConfig.Fullscreen
	ebiten.SetFullscreen(isFullscreen)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

package utils

import (
	"image/color"
	"physiGo/config"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TitleDraw(screen *ebiten.Image) {
	//Lettere 82 spazio 21 bordere 14

	// Draw Options

	//var X float32 = 21     // Starting X position
	var Y float32 = float32(config.GlobalConfig.ScreenHeight / 8)       // Y of the letters
	var border float32 = float32(config.GlobalConfig.ScreenHeight / 48) // Border size
	var space float32 = float32(config.GlobalConfig.ScreenHeight / 14)  // Space between letters
	var letterHeight float32 = float32(config.GlobalConfig.ScreenHeight / 6)
	var letterWidth float32 = float32(config.GlobalConfig.ScreenHeight / 9)

	var titleColor1 = color.RGBA{77, 153, 255, 255}
	var titleColor2 = color.RGBA{0, 255, 255, 255}

	totalWidth := space*8 + letterWidth*7
	startX := (float32(config.GlobalConfig.ScreenWidth) - totalWidth) / 2
	baseX := func(index int) float32 {
		return startX + space*float32(index+1) + letterWidth*float32(index)
	}

	// Draw "P"
	{
		x := baseX(0)
		vector.DrawFilledRect(screen,
			x, Y, border, letterHeight,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x, Y, letterWidth, border,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight/20*9, letterWidth, border,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth-border, Y, border, letterHeight/20*9,
			titleColor1, false,
		)
	}
	// Draw "H"
	{
		x := baseX(1)
		vector.DrawFilledRect(screen,
			x, Y, border, letterHeight,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth-border, Y, border, letterHeight,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight/2-border/2, letterWidth, border,
			titleColor2, false,
		)
	}
	// Draw "Y"
	{
		x := baseX(2)
		midX := x + letterWidth/2
		midY := Y + letterHeight/2
		vector.StrokeLine(screen,
			x, Y,
			midX, midY,
			border, titleColor1, false,
		)
		vector.StrokeLine(screen,
			x+letterWidth, Y,
			midX, midY,
			border, titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			midX-border/2, midY, border, letterHeight/2,
			titleColor1, false,
		)
	}
	// Draw "S"
	{
		x := baseX(3)
		vector.DrawFilledRect(screen,
			x, Y, letterWidth, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight/2-border/2, letterWidth, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight-border, letterWidth, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y, border, letterHeight/2,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth-border, Y+letterHeight/2, border, letterHeight/2,
			titleColor2, false,
		)
	}
	// Draw "I"
	{
		x := baseX(4)
		vector.DrawFilledRect(screen,
			x, Y, letterWidth, border,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight-border, letterWidth, border,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth/2-border/2, Y, border, letterHeight,
			titleColor1, false,
		)
	}
	// Draw "G"
	{
		x := baseX(5)
		vector.DrawFilledRect(screen,
			x, Y, border, letterHeight,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y, letterWidth, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight-border, letterWidth, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth/2, Y+letterHeight/2-border/2, letterWidth/2, border,
			titleColor2, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth-border, Y+letterHeight/2-border/2, border, letterHeight/2,
			titleColor2, false,
		)
	}
	// Draw "O"
	{
		x := baseX(6)
		vector.DrawFilledRect(screen,
			x, Y, border, letterHeight,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x, Y, letterWidth, border,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x+letterWidth-border, Y, border, letterHeight,
			titleColor1, false,
		)
		vector.DrawFilledRect(screen,
			x, Y+letterHeight-border, letterWidth, border,
			titleColor1, false,
		)
	}
	// Draw title
	ScreenDraw(-3, float64(config.GlobalConfig.ScreenWidth)/3.6, float64(Y+letterHeight+letterHeight/2), "sky blue", screen, "by IncredibleLego")
}

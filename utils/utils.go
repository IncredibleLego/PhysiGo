package utils

import (
	"bytes"
	_ "embed"
	"log"
	"physiGo/config"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// This file contains utility functions for the game.

//go:embed PressStart2P-Regular.ttf
var pressStart2P []byte

//go:embed LibertinusMath-Regular.ttf
var libertinusMath []byte

var (
	pressStart2PFaceOnce     sync.Once
	pressStart2PFaceSource   *text.GoTextFaceSource
	libertinusMathFaceOnce   sync.Once
	libertinusMathFaceSource *text.GoTextFaceSource
)

func getPressStart2PFaceSource() *text.GoTextFaceSource {
	pressStart2PFaceOnce.Do(func() {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(pressStart2P))
		if err != nil {
			log.Fatal(err)
		}
		pressStart2PFaceSource = s
	})
	return pressStart2PFaceSource
}

func getLibertinusMathFaceSource() *text.GoTextFaceSource {
	libertinusMathFaceOnce.Do(func() {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(libertinusMath))
		if err != nil {
			log.Fatal(err)
		}
		libertinusMathFaceSource = s
	})
	return libertinusMathFaceSource
}

// getFontSource returns the appropriate font source based on fontName
// If fontName is empty or not recognized, returns Press2Start (default)
// If fontName is "libertinus" or "physics", returns LibertinusMath
func getFontSource(fontName string) *text.GoTextFaceSource {
	switch fontName {
	case "libertinus", "physics":
		return getLibertinusMathFaceSource()
	default:
		return getPressStart2PFaceSource()
	}
}

func ScreenDraw(size float64, x, y float64, colorName string, screen *ebiten.Image, line string, fontName ...string) {
	font := ""
	if len(fontName) > 0 {
		font = fontName[0]
	}

	textFace := &text.GoTextFace{
		Source: getFontSource(font),
		Size:   config.GlobalConfig.TextDimension + size,
	}

	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(x, y)
	r, g, b, a := Color(colorName)
	textOptions.ColorScale.Scale(r, g, b, a)
	textOptions.LineSpacing = float64(size) / 10

	text.Draw(screen, line, textFace, textOptions)
}

func MeasureText(label string, fontName ...string) (float64, float64) {
	font := ""
	if len(fontName) > 0 {
		font = fontName[0]
	}

	textFace := &text.GoTextFace{
		Source: getFontSource(font),
		Size:   config.GlobalConfig.TextDimension,
	}

	boundsX, boundsY := text.Measure(label, textFace, float64(config.GlobalConfig.TextDimension)/10)
	return boundsX, boundsY
}

// MeasureTextWithSize measures text with a specific size and optional font
func MeasureTextWithSize(label string, size float64, fontName ...string) (float64, float64) {
	font := ""
	if len(fontName) > 0 {
		font = fontName[0]
	}

	textFace := &text.GoTextFace{
		Source: getFontSource(font),
		Size:   size,
	}

	boundsX, boundsY := text.Measure(label, textFace, size/10)
	return boundsX, boundsY
}

// Gives x coord to place a message in the middle of the screen given the message and the font size
func XCentered(message string, fontSize float64) float64 {
	width := float64(len(message)) * fontSize
	x := (float64(config.GlobalConfig.ScreenWidth) / 2) - (width / 2)
	return x
}

// XCenteredWithFont gives x coord to place a message in the middle of the screen
// using MeasureText for accurate positioning with the specified font
func XCenteredWithFont(message string, fontSize float64, fontName string) float64 {
	width, _ := MeasureTextWithSize(message, fontSize, fontName)
	x := (float64(config.GlobalConfig.ScreenWidth) / 2) - (width / 2)
	return x
}

func Color(colorName string) (float32, float32, float32, float32) {
	switch colorName {
	case "white":
		// RGBA: 255, 255, 255, 255
		return 1, 1, 1, 1
	case "black":
		// RGBA: 0, 0, 0, 255
		return 0, 0, 0, 1
	case "red":
		// RGBA: 255, 0, 0, 255
		return 1, 0, 0, 1
	case "green":
		// RGBA: 0, 255, 0, 255
		return 0, 1, 0, 1
	case "yellow":
		// RGBA: 255, 255, 0, 255
		return 1, 1, 0, 1
	case "cyan":
		// RGBA: 0, 255, 255, 255
		return 0, 1, 1, 1
	case "light gray":
		// RGBA: 204, 204, 204, 255
		return 0.8, 0.8, 0.8, 1
	case "orange":
		// RGBA: 255, 128, 0, 255
		return 1, 0.5, 0, 1
	case "sky blue":
		// RGBA: 77, 153, 255, 255
		return 0.3, 0.6, 1, 1
	default:
		log.Printf("Unknown color: %s", colorName)
		return 0, 0, 0, 255 // Default to black
	}
}

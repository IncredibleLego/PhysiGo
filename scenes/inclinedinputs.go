package scenes

import (
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"physiGo/config"
	"physiGo/utils"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const maxInclinedAngle = 89.0

type inclinedInputPhase int

const (
	inclinedSelectObjectPhase inclinedInputPhase = iota
	inclinedDataPhase
)

type InclinedInputScene struct {
	phase       inclinedInputPhase
	activeField int

	objectMode InclinedObjectMode
	rotaryType InclinedRotaryType

	thetaInput   string
	muSInput     string
	muKInput     string
	muRInput     string
	massInput    string
	gravityInput string
	lengthInput  string
	hBlockInput  string
	v0Input      string
	radiusInput  string

	blockImage  *ebiten.Image
	barrelImage *ebiten.Image

	lastBlink time.Time

	validationMessage string
}

func NewInclinedInputScene() *InclinedInputScene {
	return &InclinedInputScene{}
}

func (i *InclinedInputScene) ShouldPreserveState(reason SceneChangeReason) bool {
	return false
}

func (i *InclinedInputScene) Draw(screen *ebiten.Image) {
	screen.Clear()
	if i.phase == inclinedSelectObjectPhase {
		i.drawObjectSelection(screen)
		return
	}
	if i.objectMode == InclinedObjectBlock {
		i.drawBlockInput(screen)
		return
	}
	i.drawRotaryInput(screen)
}

// drawObjectSelection disegna il menu iniziale per scegliere il tipo di corpo:
// - block solido (senza rotazione)
// - corpo rotatorio (sfera, cilindro, disco) con scelta della forma
// Mostra un'anteprima visiva e fornisce opzioni di importazione da file.
func (i *InclinedInputScene) drawObjectSelection(screen *ebiten.Image) {
	textDim := config.GlobalConfig.TextDimension
	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	centerX := sw * 0.5

	title := "INCLINED PLANE - BODY TYPE"
	utils.ScreenDraw(0, utils.XCenteredWithFont(title, textDim, "libertinus"), sh*0.1, "yellow", screen, title, "libertinus")

	modeLine := "Movimento: Blocco Solido"
	if i.objectMode == InclinedObjectRotary {
		modeLine = "Movimento: Rotatorio"
	}
	utils.ScreenDraw(0, utils.XCenteredWithFont(modeLine, textDim, "libertinus"), sh*0.22, "cyan", screen, modeLine, "libertinus")

	previewX := centerX - sw*0.12
	previewY := sh * 0.38
	previewSize := sh * 0.2

	if i.objectMode == InclinedObjectBlock {
		i.drawPreviewImage(screen, i.blockImage, previewX, previewY, previewSize)
		label := "Modello: blocco senza rotazione"
		utils.ScreenDraw(-(textDim / 6), previewX+previewSize+24, previewY+previewSize*0.55, "white", screen, label, "libertinus")
	} else {
		i.drawPreviewImage(screen, i.barrelImage, previewX, previewY, previewSize)
		rotaryLabel := "Corpo: " + rotaryTypeLabel(i.rotaryType)
		formula := rotaryInertiaFormula(i.rotaryType)
		utils.ScreenDraw(-(textDim / 8), previewX+previewSize+24, previewY+previewSize*0.35, "cyan", screen, rotaryLabel, "libertinus")
		utils.ScreenDraw(-(textDim / 6), previewX+previewSize+24, previewY+previewSize*0.65, "white", screen, formula, "libertinus")

		upDown := "Up/Down per cambiare forma"
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(upDown, smallText, "libertinus"), sh*0.68, "light gray", screen, upDown, "libertinus")
	}

	helpMode := "Left/Right per cambiare"
	smallText := textDim - (textDim / 4)
	utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(helpMode, smallText, "libertinus"), sh*0.74, "light gray", screen, helpMode, "libertinus")

	status := "Invio per passare ai dati"
	utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(status, smallText, "libertinus"), sh*0.80, "light gray", screen, status, "libertinus")

	mx, my := ebiten.CursorPosition()
	importRect := inclinedImportButtonRect()
	hovered := importRect.contains(float64(mx), float64(my))
	drawInclinedImportButton(screen, hovered)

	importHint := "Clicca il bottone o premi I per importare"
	utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(importHint, smallText, "libertinus"), sh*0.96, "light gray", screen, importHint, "libertinus")

	if i.validationMessage != "" {
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(i.validationMessage, smallText, "libertinus"), sh*0.84, "red", screen, i.validationMessage, "libertinus")
	}
}

// drawPreviewImage disegna un'immagine centrata e scalata proportionalmente al target size specificato. Usato per mostrare l'anteprima block/barrel.
func (i *InclinedInputScene) drawPreviewImage(screen *ebiten.Image, img *ebiten.Image, x, y, targetSize float64) {
	if img == nil {
		return
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	if imgW <= 0 || imgH <= 0 {
		return
	}
	scale := targetSize / math.Max(imgW, imgH)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	dw := imgW * scale
	dh := imgH * scale
	op.GeoM.Translate(x+(targetSize-dw)/2, y+(targetSize-dh)/2)
	screen.DrawImage(img, op)
}

// drawBlockInput disegna i campi di input specifici per un corpo block (senza rotazione): massa, angolo, lunghezza, altezza blocco, attrito statico/dinamico, velocità iniziale, gravità.
func (i *InclinedInputScene) drawBlockInput(screen *ebiten.Image) {
	textDim := config.GlobalConfig.TextDimension
	spacing := textDim * 1.5
	startY := float64(config.GlobalConfig.ScreenHeight) * 0.23

	title := "INCLINED PLANE SETUP"
	utils.ScreenDraw(0, utils.XCenteredWithFont(title, textDim, "libertinus"), startY-textDim*1.2, "yellow", screen, title, "libertinus")

	lines := []string{
		"m (mass > 0): " + i.renderInputValueBlock(i.massInput, 0),
		"θ (0°-89°): " + i.renderInputValueBlock(i.thetaInput, 1),
		"L (length > 0): " + i.renderInputValueBlock(i.lengthInput, 2),
		"h_block (height): " + i.renderInputValueBlock(i.hBlockInput, 3),
		"μ_s (>=0): " + i.renderInputValueBlock(i.muSInput, 4),
		"μ_k (>=0): " + i.renderInputValueBlock(i.muKInput, 5),
		"v0 (initial speed): " + i.renderInputValueBlock(i.v0Input, 6),
		"g (gravity): " + i.renderInputValueBlock(i.gravityInput, 7),
	}

	for idx, line := range lines {
		color := "white"
		if idx == i.activeField {
			color = "cyan"
		}
		y := startY + float64(idx)*spacing
		utils.ScreenDraw(0, utils.XCenteredWithFont(line, textDim, "libertinus"), y, color, screen, line, "libertinus")
	}

	if i.validationMessage != "" {
		y := startY + float64(len(lines))*spacing + textDim
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(i.validationMessage, smallText, "libertinus"), y, "red", screen, i.validationMessage, "libertinus")
	}

	status := "Use arrows to move, Enter to confirm"
	prevValidation := i.validationMessage
	if i.allInputsValidBlock() {
		status = "Values ready - press Enter to continue"
	}
	i.validationMessage = prevValidation
	y := startY + float64(len(lines))*spacing + textDim*2.2
	smallText := textDim - (textDim / 3)
	utils.ScreenDraw(-(textDim / 3), utils.XCenteredWithFont(status, smallText, "libertinus"), y, "light gray", screen, status, "libertinus")
}

// drawRotaryInput disegna i campi di input specifici per un corpo rotatorio: massa, raggio, angolo, lunghezza, altezza blocco, attrito di rotolamento, velocità iniziale, gravità.
func (i *InclinedInputScene) drawRotaryInput(screen *ebiten.Image) {
	textDim := config.GlobalConfig.TextDimension
	spacing := textDim * 1.5
	startY := float64(config.GlobalConfig.ScreenHeight) * 0.23
	optionsStartY := startY + textDim*0.55

	title := "INCLINED PLANE SETUP"
	utils.ScreenDraw(0, utils.XCenteredWithFont(title, textDim, "libertinus"), startY-textDim*1.8, "yellow", screen, title, "libertinus")

	modeLine := "Corpo: " + rotaryTypeLabel(i.rotaryType)
	utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(modeLine, textDim-(textDim/4), "libertinus"), startY-textDim*0.35, "cyan", screen, modeLine, "libertinus")

	lines := []string{
		"m (mass > 0): " + i.renderInputValueRotary(i.massInput, 0),
		"r (radius > 0): " + i.renderInputValueRotary(i.radiusInput, 1),
		"θ (0°-89°): " + i.renderInputValueRotary(i.thetaInput, 2),
		"L (length > 0): " + i.renderInputValueRotary(i.lengthInput, 3),
		"h_block (height): " + i.renderInputValueRotary(i.hBlockInput, 4),
		"μ_r (>=0): " + i.renderInputValueRotary(i.muRInput, 5),
		"v0 (initial speed): " + i.renderInputValueRotary(i.v0Input, 6),
		"g (gravity): " + i.renderInputValueRotary(i.gravityInput, 7),
	}

	for idx, line := range lines {
		color := "white"
		if idx == i.activeField {
			color = "cyan"
		}
		y := optionsStartY + float64(idx)*spacing
		utils.ScreenDraw(0, utils.XCenteredWithFont(line, textDim, "libertinus"), y, color, screen, line, "libertinus")
	}

	if i.validationMessage != "" {
		y := optionsStartY + float64(len(lines))*spacing + textDim
		smallText := textDim - (textDim / 4)
		utils.ScreenDraw(-(textDim / 4), utils.XCenteredWithFont(i.validationMessage, smallText, "libertinus"), y, "red", screen, i.validationMessage, "libertinus")
	}

	status := "Use arrows to move, Enter to confirm"
	prevValidation := i.validationMessage
	if i.allInputsValidRotary() {
		status = "Values ready - press Enter to continue"
	}
	i.validationMessage = prevValidation
	y := optionsStartY + float64(len(lines))*spacing + textDim*2.2
	smallText := textDim - (textDim / 3)
	utils.ScreenDraw(-(textDim / 3), utils.XCenteredWithFont(status, smallText, "libertinus"), y, "light gray", screen, status, "libertinus")
}

func (i *InclinedInputScene) FirstLoad() {
	i.phase = inclinedSelectObjectPhase
	i.activeField = 0
	i.objectMode = InclinedObjectBlock
	i.rotaryType = RotaryDisk
	i.thetaInput = ""
	i.muSInput = "0"
	i.muKInput = "0"
	i.muRInput = "0"
	i.massInput = ""
	i.gravityInput = "9.8"
	i.lengthInput = ""
	i.hBlockInput = ""
	i.v0Input = "0"
	i.radiusInput = ""
	i.lastBlink = time.Now()
	i.validationMessage = ""
	i.loadPreviewImages()
}

// loadPreviewImages carica le immagini PNG per l'anteprima visuale dei corpi (block e barrel). Se il caricamento fallisce, l'immagine rimane nil.
func (i *InclinedInputScene) loadPreviewImages() {
	load := func(path string) *ebiten.Image {
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		decoded, _, err := image.Decode(file)
		if err != nil {
			return nil
		}
		return ebiten.NewImageFromImage(decoded)
	}
	i.blockImage = load("img/block.png")
	i.barrelImage = load("img/barrel.png")
}

func (i *InclinedInputScene) OnEnter() {}
func (i *InclinedInputScene) OnExit()  {}

// Update gestisce gli input da tastiera e mouse e aggiorna la scena:
// - nella fase selezione: routing a updateObjectSelection
// - nella fase dati: navigazione campi (frecce), conferma (Enter), input numerico
func (i *InclinedInputScene) Update() SceneId {
	if i.phase == inclinedSelectObjectPhase {
		return i.updateObjectSelection()
	}

	fieldCount := i.currentFieldCount()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		i.activeField--
		if i.activeField < 0 {
			i.activeField = fieldCount - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		i.activeField++
		if i.activeField > fieldCount-1 {
			i.activeField = 0
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if i.tryConfirmActiveField() {
			i.validationMessage = ""
			if i.activeField < fieldCount-1 {
				i.activeField++
			} else if i.allInputsValid() {
				i.storeValues()
				return InclinedPlaneSceneId
			}
		}
	}

	i.handleActiveFieldInput()
	return InclinedInputSceneId
}

// updateObjectSelection gestisce la selezione iniziale del tipo di corpo e l'importazione da file.
// Consente cambio block<->rotario (frecce sinistra/destra) e cambio forma rotaria (frecce su/giù).
func (i *InclinedInputScene) updateObjectSelection() SceneId {
	mx, my := ebiten.CursorPosition()
	importRect := inclinedImportButtonRect()
	importClicked := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && importRect.contains(float64(mx), float64(my))

	if importClicked || inpututil.IsKeyJustPressed(ebiten.KeyI) {
		err := importInclinedPlaneProblemFromFile()
		if err == nil {
			i.validationMessage = ""
			return InclinedPlaneSceneId
		}
		if errors.Is(err, errImportCancelled) {
			return InclinedInputSceneId
		}
		i.validationMessage = err.Error()
		return InclinedInputSceneId
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		if i.objectMode == InclinedObjectBlock {
			i.objectMode = InclinedObjectRotary
		} else {
			i.objectMode = InclinedObjectBlock
		}
	}

	if i.objectMode == InclinedObjectRotary {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			i.rotateRotaryType(-1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			i.rotateRotaryType(1)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		i.phase = inclinedDataPhase
		i.activeField = 0
		i.validationMessage = ""
	}

	return InclinedInputSceneId
}

// rotateRotaryType ruota il tipo di corpo rotatorio nel senso indicato da delta tra i tipi disponibili (RotarySolidCylinder, RotaryDisk, RotarySphere, etc.).
func (i *InclinedInputScene) rotateRotaryType(delta int) {
	idx := 0
	for k, kind := range rotaryTypes {
		if kind == i.rotaryType {
			idx = k
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = len(rotaryTypes) - 1
	}
	if idx >= len(rotaryTypes) {
		idx = 0
	}
	i.rotaryType = rotaryTypes[idx]
}

// currentFieldCount restituisce il numero di campi attivi per la fase di input dati (8).
func (i *InclinedInputScene) currentFieldCount() int {
	return 8
}

// renderInputValueBlock formatta il valore di input per il corpo block con le unità di misura corrette
// (es. "1.5 kg" per massa, "30°" per angolo).
func (i *InclinedInputScene) renderInputValueBlock(value string, fieldIndex int) string {
	units := map[int]string{
		0: " kg",
		1: "°",
		2: " m",
		3: " m",
		6: " m/s",
		7: " m/s^2",
	}
	return i.renderInputValueWithUnits(value, fieldIndex, units)
}

// renderInputValueRotary formatta il valore di input per il corpo rotatorio con le unità di misura corrette.
func (i *InclinedInputScene) renderInputValueRotary(value string, fieldIndex int) string {
	units := map[int]string{
		0: " kg",
		1: " m",
		2: "°",
		3: " m",
		4: " m",
		6: " m/s",
		7: " m/s^2",
	}
	return i.renderInputValueWithUnits(value, fieldIndex, units)
}

// renderInputValueWithUnits applica le unità di misura al valore e aggiunge effetto blinking se il campo è attivo (cioè in fase di editing).
func (i *InclinedInputScene) renderInputValueWithUnits(value string, fieldIndex int, units map[int]string) string {
	if fieldIndex != i.activeField {
		if value == "" {
			return "-"
		}
		return value + units[fieldIndex]
	}
	return renderBlinkingValue(&i.lastBlink, value)
}

// handleActiveFieldInput processa i caratteri digitati dall'utente per il campo attualmente selezionato.
// Indirizza il layout corretto (block vs rotario) e chiama handleNumericInput per ogni campo.
func (i *InclinedInputScene) handleActiveFieldInput() {
	if i.objectMode == InclinedObjectRotary {
		switch i.activeField {
		case 0:
			i.handleNumericInput(&i.massInput)
		case 1:
			i.handleNumericInput(&i.radiusInput)
		case 2:
			i.handleNumericInput(&i.thetaInput)
		case 3:
			i.handleNumericInput(&i.lengthInput)
		case 4:
			i.handleNumericInput(&i.hBlockInput)
		case 5:
			i.handleNumericInput(&i.muRInput)
		case 6:
			i.handleNumericInput(&i.v0Input)
		case 7:
			i.handleNumericInput(&i.gravityInput)
		}
		return
	}

	switch i.activeField {
	case 0:
		i.handleNumericInput(&i.massInput)
	case 1:
		i.handleNumericInput(&i.thetaInput)
	case 2:
		i.handleNumericInput(&i.lengthInput)
	case 3:
		i.handleNumericInput(&i.hBlockInput)
	case 4:
		i.handleNumericInput(&i.muSInput)
	case 5:
		i.handleNumericInput(&i.muKInput)
	case 6:
		i.handleNumericInput(&i.v0Input)
	case 7:
		i.handleNumericInput(&i.gravityInput)
	}
}

// handleNumericInput gestisce l'input di caratteri numerici (0-9, ., -). Limita la lunghezza a 8 caratteri e filtra input non validi.
func (i *InclinedInputScene) handleNumericInput(input *string) {
	handleNumericTextInput(input, 8)
}

// tryConfirmActiveField valida il valore nel campo attivo e, se valido, consente il passaggio al campo successivo. Diversa logica per block vs rotario.
func (i *InclinedInputScene) tryConfirmActiveField() bool {
	if i.objectMode == InclinedObjectRotary {
		return i.tryConfirmActiveFieldRotary()
	}
	return i.tryConfirmActiveFieldBlock()
}

// tryConfirmActiveFieldBlock valida il campo attivo nel layout block (massa, angolo, lunghezza, etc.) e mostra messaggi di errore specifici se la convalida fallisce.
func (i *InclinedInputScene) tryConfirmActiveFieldBlock() bool {
	switch i.activeField {
	case 0:
		_, ok, _ := parseOptionalMin(i.massInput, 0)
		if !ok {
			i.validationMessage = "m must be greater than 0"
			return false
		}
	case 1:
		_, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if !ok {
			i.validationMessage = "θ must be between 0 and 89"
			return false
		}
	case 2:
		_, ok, _ := parseOptionalMin(i.lengthInput, 0)
		if !ok {
			i.validationMessage = "L must be greater than 0"
			return false
		}
	case 3:
		if !i.validateHBlock() {
			return false
		}
	case 4:
		_, ok, _ := parseOptionalNonNegative(i.muSInput)
		if !ok {
			i.validationMessage = "μ_s must be >= 0"
			return false
		}
	case 5:
		_, ok, _ := parseOptionalNonNegative(i.muKInput)
		if !ok {
			i.validationMessage = "μ_k must be >= 0"
			return false
		}
	case 6:
		_, ok, _ := parseOptionalNonNegative(i.v0Input)
		if !ok {
			i.validationMessage = "v0 must be >= 0"
			return false
		}
	case 7:
		_, ok, _ := parseOptionalMin(i.gravityInput, 0)
		if !ok {
			i.validationMessage = "g must be greater than 0"
			return false
		}
	}
	return true
}

// tryConfirmActiveFieldRotary valida il campo attivo nel layout rotario (massa, raggio, angolo, etc.).
// Richiede raggio e attrito rotazionale (obbligatori), mentre altri campi sono opzionali.
func (i *InclinedInputScene) tryConfirmActiveFieldRotary() bool {
	switch i.activeField {
	case 0:
		_, ok, _ := parseOptionalMin(i.massInput, 0)
		if !ok {
			i.validationMessage = "m must be greater than 0"
			return false
		}
	case 1:
		if strings.TrimSpace(i.radiusInput) == "" {
			i.validationMessage = "r is required"
			return false
		}
		_, ok := parseRequiredMin(i.radiusInput, 0)
		if !ok {
			i.validationMessage = "r must be greater than 0"
			return false
		}
	case 2:
		_, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if !ok {
			i.validationMessage = "θ must be between 0 and 89"
			return false
		}
	case 3:
		_, ok, _ := parseOptionalMin(i.lengthInput, 0)
		if !ok {
			i.validationMessage = "L must be greater than 0"
			return false
		}
	case 4:
		if !i.validateHBlock() {
			return false
		}
	case 5:
		_, ok := parseRequiredNonNegative(i.muRInput)
		if !ok {
			i.validationMessage = "μ_r must be >= 0"
			return false
		}
	case 6:
		_, ok, _ := parseOptionalNonNegative(i.v0Input)
		if !ok {
			i.validationMessage = "v0 must be >= 0"
			return false
		}
	case 7:
		_, ok, _ := parseOptionalMin(i.gravityInput, 0)
		if !ok {
			i.validationMessage = "g must be greater than 0"
			return false
		}
	}
	return true
}

// allInputsValid controlla se TUTTI i campi sono validi per passare alla simulazione. Indirizza a allInputsValidBlock o allInputsValidRotary a seconda del tipo di corpo.
func (i *InclinedInputScene) allInputsValid() bool {
	if i.objectMode == InclinedObjectRotary {
		return i.allInputsValidRotary()
	}
	return i.allInputsValidBlock()
}

// allInputsValidBlock valida TUTTI i campi per il corpo block:
// massa/lunghezza/gravità obbligatori, attrito angolo opzionali,
// richiede L o h_block (almeno uno), e coerenza geometrica tra h_block e L*sin(θ).
func (i *InclinedInputScene) allInputsValidBlock() bool {
	if _, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.massInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.muSInput); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.muKInput); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.gravityInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.lengthInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.v0Input); !ok {
		return false
	}
	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) == "" {
		i.validationMessage = "Insert L or h_block"
		return false
	}
	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) != "" {
		theta, _ := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if theta <= 0 {
			i.validationMessage = "θ must be > 0 if only h_block is used"
			return false
		}
	}
	if !i.validateHBlock() {
		return false
	}
	return true
}

// allInputsValidRotary valida TUTTI i campi per il corpo rotatorio:
// massa/raggio/angolo/attrito_rotazionale obbligatori, altri opzionali,
// richiede L o h_block, e coerenza geometrica.
func (i *InclinedInputScene) allInputsValidRotary() bool {
	if _, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle); !ok {
		return false
	}
	if _, ok := parseRequiredNonNegative(i.muRInput); !ok {
		i.validationMessage = "μ_r must be >= 0"
		return false
	}
	if _, ok, _ := parseOptionalMin(i.massInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.gravityInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalMin(i.lengthInput, 0); !ok {
		return false
	}
	if _, ok, _ := parseOptionalNonNegative(i.v0Input); !ok {
		return false
	}
	if strings.TrimSpace(i.radiusInput) == "" {
		i.validationMessage = "r is required"
		return false
	}
	if _, ok := parseRequiredMin(i.radiusInput, 0); !ok {
		i.validationMessage = "r must be greater than 0"
		return false
	}
	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) == "" {
		i.validationMessage = "Insert L or h_block"
		return false
	}
	if strings.TrimSpace(i.lengthInput) == "" && strings.TrimSpace(i.hBlockInput) != "" {
		theta, _ := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
		if theta <= 0 {
			i.validationMessage = "θ must be > 0 if only h_block is used"
			return false
		}
	}
	if !i.validateHBlock() {
		return false
	}
	return true
}

// storeValues acquisisce i valori validati da tutti i campi,
// calcola i derivati geometrici (es. L da h_block e θ), e
// sincronizza tutto nella configurazione globale per la simulazione.
func (i *InclinedInputScene) storeValues() {
	theta, _ := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
	mass, _, massSet := parseOptionalMin(i.massInput, 0)
	length, _, lengthSet := parseOptionalMin(i.lengthInput, 0)
	gravity, _, gravitySet := parseOptionalMin(i.gravityInput, 0)
	v0, _, _ := parseOptionalNonNegative(i.v0Input)
	if !gravitySet {
		gravity = 9.8
	}
	if !massSet {
		mass = 1.0
	}

	hBlock := 0.0
	if strings.TrimSpace(i.hBlockInput) != "" {
		hBlock, _ = parseFloatInput(i.hBlockInput)
	}
	if !lengthSet && hBlock > 0 {
		thetaRad := theta * math.Pi / 180.0
		sinTheta := math.Sin(thetaRad)
		if sinTheta > 0 {
			length = hBlock / sinTheta
		}
	}

	config.GlobalConfig.InclinedObjectMode = string(i.objectMode)
	config.GlobalConfig.InclinedRotaryType = string(i.rotaryType)
	config.GlobalConfig.InclinedTheta = theta
	config.GlobalConfig.InclinedMass = mass
	config.GlobalConfig.InclinedLength = length
	config.GlobalConfig.InclinedHBlock = hBlock
	config.GlobalConfig.InclinedGravity = gravity
	config.GlobalConfig.InclinedInitialVelocity = v0
	config.GlobalConfig.InclinedGravitySet = gravitySet

	if i.objectMode == InclinedObjectRotary {
		radius, _ := parseRequiredMin(i.radiusInput, 0)
		muR, _ := parseRequiredNonNegative(i.muRInput)
		config.GlobalConfig.InclinedRadius = radius
		config.GlobalConfig.InclinedMuR = muR
		config.GlobalConfig.InclinedMuRSet = true
		config.GlobalConfig.InclinedMuS = 0
		config.GlobalConfig.InclinedMuK = 0
		config.GlobalConfig.InclinedMuSSet = false
		config.GlobalConfig.InclinedMuKSet = false
		return
	}

	muS, _, muSSet := parseOptionalNonNegative(i.muSInput)
	muK, _, muKSet := parseOptionalNonNegative(i.muKInput)
	config.GlobalConfig.InclinedRadius = 0
	config.GlobalConfig.InclinedMuR = 0
	config.GlobalConfig.InclinedMuRSet = false
	config.GlobalConfig.InclinedMuS = muS
	config.GlobalConfig.InclinedMuK = muK
	config.GlobalConfig.InclinedMuSSet = muSSet
	config.GlobalConfig.InclinedMuKSet = muKSet
}

// parseRequiredRange valida che l'input non sia vuoto e sia un numero in [min, max].
// Restituisce il valore e un booleano di validità.
func parseRequiredRange(input string, min, max float64) (float64, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false
	}
	if value < min || value > max {
		return 0, false
	}
	return value, true
}

// parseRequiredMin valida che l'input non sia vuoto e sia un numero > min.
func parseRequiredMin(input string, min float64) (float64, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false
	}
	if value <= min {
		return 0, false
	}
	return value, true
}

// parseRequiredNonNegative valida che l'input non sia vuoto e sia un numero >= 0.
func parseRequiredNonNegative(input string) (float64, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false
	}
	if value < 0 {
		return 0, false
	}
	return value, true
}

// parseOptionalMin permette input vuoto (che restituisce isSet=false).
// Se presente, valida che sia un numero > min. Restituisce (valore, ok, isSet).
func parseOptionalMin(input string, min float64) (float64, bool, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, true, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false, true
	}
	if value <= min {
		return 0, false, true
	}
	return value, true, true
}

// parseOptionalNonNegative permette input vuoto (che restituisce isSet=false).
// Se presente, valida che sia un numero >= 0. Restituisce (valore, ok, isSet).
func parseOptionalNonNegative(input string) (float64, bool, bool) {
	if strings.TrimSpace(input) == "" {
		return 0, true, false
	}
	value, err := parseFloatInput(input)
	if err != nil {
		return 0, false, true
	}
	if value < 0 {
		return 0, false, true
	}
	return value, true, true
}

// parseFloatInput converte una stringa in float64, accettando "," come separatore decimale
// (automaticamente convertito a ".") e applicando trim di spazi.
func parseFloatInput(input string) (float64, error) {
	clean := strings.ReplaceAll(strings.TrimSpace(input), ",", ".")
	return strconv.ParseFloat(clean, 64)
}

// validateHBlock valida il campo altezza blocco (h_block):
// - se vuoto, accettato per default
// - se presente, deve essere > 0 e <= L*sin(θ) (coerenza geometrica)
// - richiede che theta e L (o almeno theta) siano già inseriti e validi.
func (i *InclinedInputScene) validateHBlock() bool {
	if strings.TrimSpace(i.hBlockInput) == "" {
		return true
	}

	hBlock, err := parseFloatInput(i.hBlockInput)
	if err != nil || hBlock <= 0 {
		i.validationMessage = "h_block must be greater than 0"
		return false
	}

	thetaStr := strings.TrimSpace(i.thetaInput)
	lengthStr := strings.TrimSpace(i.lengthInput)
	if thetaStr == "" {
		i.validationMessage = "Enter theta first"
		return false
	}

	theta, ok := parseRequiredRange(i.thetaInput, 0, maxInclinedAngle)
	if !ok {
		i.validationMessage = "Enter valid theta first"
		return false
	}

	if lengthStr == "" {
		if theta <= 0 {
			i.validationMessage = "theta must be > 0 if L is empty"
			return false
		}
		return true
	}

	length, ok := parseRequiredMin(i.lengthInput, 0)
	if !ok {
		i.validationMessage = "Enter valid L first"
		return false
	}

	thetaRad := theta * math.Pi / 180.0
	hMax := length * math.Sin(thetaRad)
	const epsilon = 0.01
	if hBlock > hMax+epsilon {
		i.validationMessage = "h_block must be <= L*sin(θ) = " + strconv.FormatFloat(hMax, 'f', 2, 64) + " m"
		return false
	}

	return true
}

var _ Scene = (*InclinedInputScene)(nil)

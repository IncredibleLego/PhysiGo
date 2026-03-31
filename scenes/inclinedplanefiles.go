package scenes

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"physiGo/config"
	"physiGo/utils"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var errImportCancelled = errors.New("import cancelled")

type inclinedImportedBlockProblem struct {
	Mass         float64
	Angle        float64
	Length       float64
	Height       float64
	StaticCoeff  float64
	DynamicCoeff float64
	V0           float64
	Gravity      float64
}

type inclinedImportedRotaryProblem struct {
	Body            InclinedRotaryType
	Mass            float64
	Radius          float64
	Angle           float64
	Length          float64
	Height          float64
	RotationalCoeff float64
	V0              float64
	Gravity         float64
}

// inclinedImportButtonRect calcola posizione e dimensioni del pulsante
// "IMPORTA DA FILE" in base alla risoluzione corrente.
func inclinedImportButtonRect() uiRect {
	sw := float64(config.GlobalConfig.ScreenWidth)
	sh := float64(config.GlobalConfig.ScreenHeight)
	textDim := config.GlobalConfig.TextDimension

	buttonW := sw * 0.33
	buttonH := textDim * 1.45
	x := (sw - buttonW) / 2
	y := sh * 0.61

	return uiRect{x: x, y: y, w: buttonW, h: buttonH}
}

// drawInclinedImportButton disegna il pulsante di import con colore diverso quando il mouse e in hover.
func drawInclinedImportButton(screen *ebiten.Image, hovered bool) {
	rect := inclinedImportButtonRect()
	fillColor := color.RGBA{34, 74, 114, 255}
	if hovered {
		fillColor = color.RGBA{45, 98, 149, 255}
	}

	vector.DrawFilledRect(screen, float32(rect.x), float32(rect.y), float32(rect.w), float32(rect.h), fillColor, false)
	label := "IMPORTA DA FILE"
	labelSize := config.GlobalConfig.TextDimension * 0.62
	labelW, _ := utils.MeasureTextWithSize(label, labelSize, "libertinus")
	labelX := rect.x + (rect.w-labelW)/2
	labelY := rect.y + rect.h*0.42
	utils.ScreenDraw(labelSize-config.GlobalConfig.TextDimension, labelX, labelY, "white", screen, label, "libertinus")
}

// importInclinedPlaneProblemFromFile gestisce tutto il flusso di importazione:
// selezione file, lettura JSON, riconoscimento tipo problema e applicazione
// dei valori nella configurazione globale.
func importInclinedPlaneProblemFromFile() error {
	path, err := pickInclinedJSONFile()
	if err != nil {
		return err
	}

	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return errors.New("tipo di file errato")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("impossibile leggere il file: %w", err)
	}

	kind, payload, err := decodeInclinedProblemPayload(data)
	if err != nil {
		return err
	}

	if kind == InclinedObjectBlock {
		problem, err := parseInclinedBlockPayload(payload)
		if err != nil {
			return err
		}
		applyImportedInclinedBlock(problem)
		return nil
	}

	problem, err := parseInclinedRotaryPayload(payload)
	if err != nil {
		return err
	}
	applyImportedInclinedRotary(problem)
	return nil
}

// pickInclinedJSONFile apre un file-picker (zenity) e restituisce il path scelto.
// Distingue annullamento utente, comando mancante e altri errori di esecuzione.
func pickInclinedJSONFile() (string, error) {
	startDir := "examples/"
	if wd, err := os.Getwd(); err == nil {
		startDir = filepath.Join(wd, "examples") + string(os.PathSeparator)
	}

	path, err := pickJSONFileByOS(startDir)
	if err != nil {
		return "", err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return "", errImportCancelled
	}

	return path, nil
}

// decodeInclinedProblemPayload effettua il parsing base del JSON e deduce il tipo
// di problema (block o rotary) in base ai campi presenti.
func decodeInclinedProblemPayload(data []byte) (InclinedObjectMode, map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", nil, errors.New("json non valido")
	}

	if payload == nil {
		return "", nil, errors.New("json non valido")
	}

	if _, hasBody := payload["body"]; hasBody {
		return InclinedObjectRotary, payload, nil
	}

	if _, hasRadius := payload["radius"]; hasRadius {
		return "", nil, errors.New("campo body mancante per il problema rotatorio")
	}
	if _, hasRotCoeff := payload["rotational_coeff"]; hasRotCoeff {
		return "", nil, errors.New("campo body mancante per il problema rotatorio")
	}

	return InclinedObjectBlock, payload, nil
}

// parseInclinedBlockPayload valida e converte il payload JSON del problema block
// in una struttura tipizzata pronta per essere applicata alla config.
func parseInclinedBlockPayload(payload map[string]interface{}) (*inclinedImportedBlockProblem, error) {
	required := []string{"mass", "angle", "length", "height", "static_coeff", "dynamic_coeff", "v0", "gravity"}
	allowed := map[string]struct{}{
		"mass":          {},
		"angle":         {},
		"length":        {},
		"height":        {},
		"static_coeff":  {},
		"dynamic_coeff": {},
		"v0":            {},
		"gravity":       {},
	}

	if err := validateAllowedAndRequiredKeys(payload, required, allowed); err != nil {
		return nil, err
	}

	mass, err := readNumberField(payload, "mass")
	if err != nil {
		return nil, err
	}
	angle, err := readNumberField(payload, "angle")
	if err != nil {
		return nil, err
	}
	length, err := readNumberField(payload, "length")
	if err != nil {
		return nil, err
	}
	height, err := readNumberField(payload, "height")
	if err != nil {
		return nil, err
	}
	muS, err := readNumberField(payload, "static_coeff")
	if err != nil {
		return nil, err
	}
	muK, err := readNumberField(payload, "dynamic_coeff")
	if err != nil {
		return nil, err
	}
	v0, err := readNumberField(payload, "v0")
	if err != nil {
		return nil, err
	}
	g, err := readNumberField(payload, "gravity")
	if err != nil {
		return nil, err
	}

	if err := validateInclinedCommonValues(mass, angle, length, height, v0, g); err != nil {
		return nil, err
	}
	if muS < 0 {
		return nil, errors.New("static_coeff deve essere ≥ 0")
	}
	if muK < 0 {
		return nil, errors.New("dynamic_coeff deve essere ≥ 0")
	}
	if muS > 0 && muK > muS {
		return nil, errors.New("dynamic_coeff deve essere <= static_coeff quando static_coeff > 0")
	}

	return &inclinedImportedBlockProblem{
		Mass:         mass,
		Angle:        angle,
		Length:       length,
		Height:       height,
		StaticCoeff:  muS,
		DynamicCoeff: muK,
		V0:           v0,
		Gravity:      g,
	}, nil
}

// parseInclinedRotaryPayload valida e converte il payload JSON del problema rotatorio,
// includendo tipo corpo, raggio e coefficiente rotazionale.
func parseInclinedRotaryPayload(payload map[string]interface{}) (*inclinedImportedRotaryProblem, error) {
	required := []string{"body", "mass", "radius", "angle", "length", "height", "rotational_coeff", "v0", "gravity"}
	allowed := map[string]struct{}{
		"body":             {},
		"mass":             {},
		"radius":           {},
		"angle":            {},
		"length":           {},
		"height":           {},
		"rotational_coeff": {},
		"v₀":               {},
		"gravity":          {},
	}

	if err := validateAllowedAndRequiredKeys(payload, required, allowed); err != nil {
		return nil, err
	}

	bodyValue, ok := payload["body"]
	if !ok {
		return nil, errors.New("campo mancante: body")
	}
	body, err := parseRotaryBodyType(bodyValue)
	if err != nil {
		return nil, err
	}

	mass, err := readNumberField(payload, "mass")
	if err != nil {
		return nil, err
	}
	radius, err := readNumberField(payload, "radius")
	if err != nil {
		return nil, err
	}
	angle, err := readNumberField(payload, "angle")
	if err != nil {
		return nil, err
	}
	length, err := readNumberField(payload, "length")
	if err != nil {
		return nil, err
	}
	height, err := readNumberField(payload, "height")
	if err != nil {
		return nil, err
	}
	muR, err := readNumberField(payload, "rotational_coeff")
	if err != nil {
		return nil, err
	}
	v0, err := readNumberField(payload, "v0")
	if err != nil {
		return nil, err
	}
	g, err := readNumberField(payload, "gravity")
	if err != nil {
		return nil, err
	}

	if err := validateInclinedCommonValues(mass, angle, length, height, v0, g); err != nil {
		return nil, err
	}
	if radius <= 0 {
		return nil, errors.New("radius deve essere > 0")
	}
	if muR < 0 {
		return nil, errors.New("rotational_coeff deve essere >= 0")
	}

	return &inclinedImportedRotaryProblem{
		Body:            body,
		Mass:            mass,
		Radius:          radius,
		Angle:           angle,
		Length:          length,
		Height:          height,
		RotationalCoeff: muR,
		V0:              v0,
		Gravity:         g,
	}, nil
}

// validateAllowedAndRequiredKeys controlla che il payload contenga solo i campi supportati e che tutti i campi obbligatori siano presenti.
func validateAllowedAndRequiredKeys(payload map[string]interface{}, required []string, allowed map[string]struct{}) error {
	for key := range payload {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("campo non supportato: %s", key)
		}
	}

	for _, key := range required {
		if _, ok := payload[key]; !ok {
			return fmt.Errorf("campo mancante: %s", key)
		}
	}

	return nil
}

// readNumberField legge un campo numerico dal payload JSON e verifica che sia realmente un numero finito (no NaN, no infinito).
func readNumberField(payload map[string]interface{}, key string) (float64, error) {
	value, ok := payload[key]
	if !ok {
		return 0, fmt.Errorf("campo mancante: %s", key)
	}
	number, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("campo %s non numerico", key)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("campo %s non valido", key)
	}
	return number, nil
}

// readStringField legge un campo testuale non vuoto dal payload JSON.
// E utile per campi stringa quando non vengono codificati come numeri.
func readStringField(payload map[string]interface{}, key string) (string, error) {
	value, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("campo mancante: %s", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("campo %s non testuale", key)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("campo %s vuoto", key)
	}
	return text, nil
}

// validateInclinedCommonValues valida i vincoli fisici comuni tra payload block e rotary: intervalli di massa/angolo/lunghezza/altezza/gravita/velocita e coerenza h <= L*sin(theta).
func validateInclinedCommonValues(mass, angle, length, height, v0, gravity float64) error {
	if mass <= 0 {
		return errors.New("mass deve essere > 0")
	}
	if angle < 0 || angle > maxInclinedAngle {
		return errors.New("angle deve essere tra 0 e 89")
	}
	if length <= 0 {
		return errors.New("length deve essere > 0")
	}
	if height <= 0 {
		return errors.New("height deve essere > 0")
	}
	if v0 < 0 {
		return errors.New("v0 deve essere ≥ 0")
	}
	if gravity <= 0 {
		return errors.New("gravity deve essere > 0")
	}

	thetaRad := angle * math.Pi / 180
	hMax := length * math.Sin(thetaRad)
	const epsilon = 0.01
	if height > hMax+epsilon {
		return fmt.Errorf("height incongruente: deve essere <= L*sin(angle) = %.2f", hMax)
	}

	return nil
}

// parseRotaryBodyType converte il campo numerico "body" (1..5) nel relativo enum InclinedRotaryType usato internamente.
func parseRotaryBodyType(value interface{}) (InclinedRotaryType, error) {
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return "", errors.New("body deve essere un intero tra 1 e 5")
	}
	if math.Trunc(number) != number {
		return "", errors.New("body deve essere un intero tra 1 e 5")
	}

	switch int(number) {
	case 1:
		return RotaryRing, nil
	case 2:
		return RotaryDisk, nil
	case 3:
		return RotarySphere, nil
	case 4:
		return RotaryHollowCylinder, nil
	case 5:
		return RotarySolidCylinder, nil
	default:
		return "", errors.New("body non valido: usa 1=anello, 2=disco, 3=sfera, 4=cilindro_vuoto, 5=cilindro_pieno")
	}
}

// applyImportedInclinedBlock copia i dati importati del caso block nella onfigurazione globale, azzerando i campi specifici del caso rotatorio.
func applyImportedInclinedBlock(problem *inclinedImportedBlockProblem) {
	config.GlobalConfig.InclinedObjectMode = string(InclinedObjectBlock)
	config.GlobalConfig.InclinedRotaryType = ""
	config.GlobalConfig.InclinedRadius = 0
	config.GlobalConfig.InclinedMuR = 0
	config.GlobalConfig.InclinedMuRSet = false

	config.GlobalConfig.InclinedMass = problem.Mass
	config.GlobalConfig.InclinedTheta = problem.Angle
	config.GlobalConfig.InclinedLength = problem.Length
	config.GlobalConfig.InclinedHBlock = problem.Height
	config.GlobalConfig.InclinedMuS = problem.StaticCoeff
	config.GlobalConfig.InclinedMuK = problem.DynamicCoeff
	config.GlobalConfig.InclinedMuSSet = true
	config.GlobalConfig.InclinedMuKSet = true
	config.GlobalConfig.InclinedInitialVelocity = problem.V0
	config.GlobalConfig.InclinedGravity = problem.Gravity
	config.GlobalConfig.InclinedGravitySet = true
}

// applyImportedInclinedRotary copia i dati importati del caso rotatorio nella configurazione globale, disabilitando i campi di attrito del caso block.
func applyImportedInclinedRotary(problem *inclinedImportedRotaryProblem) {
	config.GlobalConfig.InclinedObjectMode = string(InclinedObjectRotary)
	config.GlobalConfig.InclinedRotaryType = string(problem.Body)
	config.GlobalConfig.InclinedRadius = problem.Radius
	config.GlobalConfig.InclinedMuR = problem.RotationalCoeff
	config.GlobalConfig.InclinedMuRSet = true

	config.GlobalConfig.InclinedMass = problem.Mass
	config.GlobalConfig.InclinedTheta = problem.Angle
	config.GlobalConfig.InclinedLength = problem.Length
	config.GlobalConfig.InclinedHBlock = problem.Height
	config.GlobalConfig.InclinedInitialVelocity = problem.V0
	config.GlobalConfig.InclinedGravity = problem.Gravity
	config.GlobalConfig.InclinedGravitySet = true

	config.GlobalConfig.InclinedMuS = 0
	config.GlobalConfig.InclinedMuK = 0
	config.GlobalConfig.InclinedMuSSet = false
	config.GlobalConfig.InclinedMuKSet = false
}

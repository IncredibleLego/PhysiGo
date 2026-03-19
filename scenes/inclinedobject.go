package scenes

// InclinedObjectMode identifica la categoria di corpo usata sul piano inclinato.
type InclinedObjectMode string

const (
	InclinedObjectBlock  InclinedObjectMode = "block"
	InclinedObjectRotary InclinedObjectMode = "rotary"
)

// InclinedRotaryType identifica il modello di corpo rigido rotatorio.
type InclinedRotaryType string

const (
	RotaryRing           InclinedRotaryType = "ring"
	RotaryDisk           InclinedRotaryType = "disk"
	RotarySphere         InclinedRotaryType = "sphere"
	RotaryHollowCylinder InclinedRotaryType = "hollow_cylinder"
	RotarySolidCylinder  InclinedRotaryType = "solid_cylinder"
)

var rotaryTypes = []InclinedRotaryType{
	RotaryRing,
	RotaryDisk,
	RotarySphere,
	RotaryHollowCylinder,
	RotarySolidCylinder,
}

func rotaryTypeLabel(kind InclinedRotaryType) string {
	switch kind {
	case RotaryRing:
		return "Anello"
	case RotaryDisk:
		return "Disco"
	case RotarySphere:
		return "Sfera"
	case RotaryHollowCylinder:
		return "Cilindro vuoto"
	case RotarySolidCylinder:
		return "Cilindro pieno"
	default:
		return "Disco"
	}
}

func rotaryInertiaFormula(kind InclinedRotaryType) string {
	switch kind {
	case RotaryRing, RotaryHollowCylinder:
		return "I = m*r^2"
	case RotaryDisk, RotarySolidCylinder:
		return "I = 1/2*m*r^2"
	case RotarySphere:
		return "I = 2/5*m*r^2"
	default:
		return "I = 1/2*m*r^2"
	}
}

// rotaryInertiaFactor restituisce il coefficiente k nella formula I = k*m*r^2.
func rotaryInertiaFactor(kind InclinedRotaryType) float64 {
	switch kind {
	case RotaryRing, RotaryHollowCylinder:
		return 1.0
	case RotaryDisk, RotarySolidCylinder:
		return 0.5
	case RotarySphere:
		return 0.4
	default:
		return 0.5
	}
}

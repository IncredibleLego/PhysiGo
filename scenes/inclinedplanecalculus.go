package scenes

import (
	"math"
	"physiGo/config"
)

// InclinedPlaneCalculus contiene tutti i parametri pre-calcolati per la simulazione del piano inclinato, derivati dalla configurazione di input
type InclinedPlaneCalculus struct {
	// Geometria di input e modello del corpo
	Theta         float64
	InitialHeight float64
	ObjectMode    InclinedObjectMode
	RotaryType    InclinedRotaryType
	Radius        float64
	MuR           float64

	// Forze
	WeightParallel     float64
	WeightPerp         float64
	Normal             float64
	StaticFrictionMax  float64
	StaticFrictionReal float64
	DynamicFriction    float64
	NetForce           float64
	Acceleration       float64
	Slides             bool

	// Modello rotazionale (I = k*m*r²)
	RotaryInertiaFactor float64
	MomentOfInertia     float64

	// Cinematica sul piano inclinato
	DistanceToBase        float64
	VelocityAtBase        float64
	TimeToBase            float64
	InitialVelocity       float64
	StopsOnIncline        bool
	StopDistanceOnIncline float64

	// Cinematica sul terreno dopo la base
	HorizontalDecel    float64
	HorizontalStopDist float64
	HorizontalStopTime float64

	// Riferimento energetico
	Mass                   float64
	Gravity                float64
	InitialKineticTrans    float64
	InitialKineticRot      float64
	InitialMechanicalTotal float64
}

// InclinedPlaneSimState rappresenta lo stato cinematico completo della simulazione in un istante dato.
type InclinedPlaneSimState struct {
	Time     float64
	S        float64 // distanza percorsa lungo il piano inclinato
	HorizS   float64 // distanza percorsa sul terreno orizzontale
	Velocity float64
	HBlock   float64 // altezza corrente del blocco
	Phase    string  // "ready", "incline", "horizontal", "stopped"

	// Dinamica rotazionale
	AngularVelocity     float64
	AngularAcceleration float64
	RotationAngle       float64

	// Energie e lavoro
	KineticTranslational float64
	KineticRotational    float64
	PotentialEnergy      float64
	TotalMechanical      float64
	TotalWork            float64

	BaseReached       bool
	BaseReachTime     float64
	BaseReachVelocity float64
	BaseReachDistance float64

	SimulationEnded     bool
	SimulationEndTime   float64
	SimulationEndHorizS float64

	Completed bool
}

// TotalDuration restituisce il tempo totale della simulazione (piano inclinato + fase orizzontale). Restituisce 0 se il corpo non scivola. Tiene conto di:
// - Arresto sul piano inclinato
// - Movimento orizzontale con attrito fino allo stop
// - Movimento orizzontale senza attrito (infinito)
func (c InclinedPlaneCalculus) TotalDuration() float64 {
	if !c.Slides {
		return 0
	}
	if c.StopsOnIncline {
		return c.TimeToBase
	}
	if c.HorizontalDecel > 0 {
		return c.TimeToBase + c.HorizontalStopTime
	}
	return c.TimeToBase
}

// SimProgress restituisce il progresso della simulazione in [0,1] normalizzato al tempo totale. Usato per aggiornare la progress bar visuale.
func (c InclinedPlaneCalculus) SimProgress(t float64) float64 {
	total := c.TotalDuration()
	if total <= 0 {
		return 0
	}
	return clamp01(t / total)
}

// BaseReachProgressFraction restituisce la frazione di tempo totale al quale il corpo raggiunge la base del piano inclinato.
// Restituisce -1 se non applicabile (es. se il corpo non scivola o si ferma sul piano).
func (c InclinedPlaneCalculus) BaseReachProgressFraction() float64 {
	if !c.Slides || c.StopsOnIncline || c.TimeToBase <= 0 {
		return -1
	}
	total := c.TotalDuration()
	if total <= 0 {
		return -1
	}
	return clamp01(c.TimeToBase / total)
}

// CurrentAcceleration restituisce l'accelerazione lineare associata alla fase simulativa corrente:
// - "incline"/"ready": accelerazione sul piano inclinato (positiva se scivola)
// - "horizontal": decelerazione (negativa) dovuta all'attrito orizzontale
// - "stopped": zero (corpo fermo)
func (c InclinedPlaneCalculus) CurrentAcceleration(phase string) float64 {
	if !c.Slides {
		return 0
	}
	switch phase {
	case "horizontal":
		if c.HorizontalDecel <= 0 {
			return 0
		}
		return -c.HorizontalDecel
	case "stopped":
		return 0
	default:
		return c.Acceleration
	}
}

// CurrentAngularAcceleration restituisce l'accelerazione angolare per il corpo rotatorio.
// È calcolata come accelerazione lineare divisa per il raggio: α = a / r.
// Ritorna 0 se il corpo non è rotatorio o se il raggio non è valido.
func (c InclinedPlaneCalculus) CurrentAngularAcceleration(phase string) float64 {
	if c.ObjectMode != InclinedObjectRotary || c.Radius <= 0 {
		return 0
	}
	a := c.CurrentAcceleration(phase)
	if c.Radius == 0 {
		return 0
	}
	return a / c.Radius
}

// DistanceFromBase restituisce la distanza rimanente fino alla base del piano inclinato.
// Calcolata come DistanceToBase - s (distanza percorsa).
// Restituisce 0 se il corpo è già sulla fase orizzontale o se ha superato la base.
func (c InclinedPlaneCalculus) DistanceFromBase(s float64, phase string) float64 {
	if phase == "incline" || phase == "ready" {
		d := c.DistanceToBase - s
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// ComputeStateAtTime calcola lo stato cinematico completo al tempo t utilizzando i valori pre-calcolati.
// Gestisce tre fasi:
// 1. READY -> INCLINE: corpo su piano inclinato, accelerazione costante
// 2. INCLINE -> HORIZONTAL: corpo raggiunge base, passa a movimento orizzontale
// 3. HORIZONTAL -> STOPPED: corpo si ferma per attrito (se HorizontalDecel > 0)
// Implementa protezione floating-point con tolleranza stopEpsilon per evitare residui numerici vicino al punto di arresto analitico.
func (c InclinedPlaneCalculus) ComputeStateAtTime(t float64) InclinedPlaneSimState {
	h0 := c.InitialHeight

	// CASO: corpo non scivola -> resta sempre fermo al tempo t=0
	if !c.Slides {
		state := InclinedPlaneSimState{
			Time:     0,
			Velocity: c.InitialVelocity,
			HBlock:   h0,
			Phase:    "ready",
		}
		return c.withEnergyAndRotation(state)
	}

	if t < 0 {
		t = 0
	}

	// Limita il tempo all'istante di arresto quando esiste un stop finito.
	total := c.TotalDuration()
	if total > 0 && c.HorizontalDecel > 0 && t > total {
		t = total
	}

	thetaRad := c.Theta * math.Pi / 180.0
	v0 := c.InitialVelocity
	a := c.Acceleration

	state := InclinedPlaneSimState{
		Time:     t,
		Velocity: v0,
		HBlock:   h0,
		Phase:    "ready",
	}

	if t == 0 {
		return c.withEnergyAndRotation(state)
	}

	inclineTime := c.TimeToBase

	// FASE 1: corpo si ferma sul piano inclinato prima di raggiunga la base.
	if c.StopsOnIncline {
		tInc := t
		if tInc > inclineTime {
			tInc = inclineTime
		}
		state.Phase = "incline"
		state.S = v0*tInc + 0.5*a*tInc*tInc
		state.Velocity = v0 + a*tInc
		if state.Velocity < 0 {
			state.Velocity = 0
		}
		if state.S < 0 {
			state.S = 0
		}
		heightLost := state.S * math.Sin(thetaRad)
		state.HBlock = h0 - heightLost
		if state.HBlock < 0 {
			state.HBlock = 0
		}
		if t >= inclineTime {
			state.S = c.StopDistanceOnIncline
			state.Velocity = 0
			state.Phase = "stopped"
			state.Completed = true
			state.SimulationEnded = true
			state.SimulationEndTime = inclineTime
			state.Time = inclineTime
		}
		return c.withEnergyAndRotation(state)
	}

	// FASE 2: corpo è ancora sul piano inclinato (non ha raggiunto la base).
	if t < inclineTime {
		state.Phase = "incline"
		state.S = v0*t + 0.5*a*t*t
		state.Velocity = v0 + a*t
		if state.Velocity < 0 {
			state.Velocity = 0
		}
		if state.S < 0 {
			state.S = 0
		}
		heightLost := state.S * math.Sin(thetaRad)
		state.HBlock = h0 - heightLost
		if state.HBlock < 0 {
			state.HBlock = 0
		}
		return c.withEnergyAndRotation(state)
	}

	// FASE 3: corpo ha raggiunto la base del piano e è su terreno orizzontale.
	state.Phase = "horizontal"
	state.S = c.DistanceToBase
	state.HBlock = 0
	state.BaseReached = true
	state.BaseReachTime = inclineTime
	state.BaseReachVelocity = c.VelocityAtBase
	state.BaseReachDistance = c.DistanceToBase

	horizontalT := t - inclineTime
	if horizontalT < 0 {
		horizontalT = 0
	}

	// Sottofase 3a: nessun attrito orizzontale -> moto uniforme infinito
	if c.HorizontalDecel <= 0 {
		state.HorizS = c.VelocityAtBase * horizontalT
		state.Velocity = c.VelocityAtBase
		return c.withEnergyAndRotation(state)
	}

	// Sottofase 3b: attrito orizzontale presente -> decelerazione fino allo stop
	vBase := c.VelocityAtBase
	decel := c.HorizontalDecel
	state.HorizS = vBase*horizontalT - 0.5*decel*horizontalT*horizontalT
	state.Velocity = vBase - decel*horizontalT

	// Protezione da residui floating-point vicino al punto di arresto analitico.
	// Usa una tolleranza piccola (stopEpsilon) per evitare che v ~= epsilon impedisca
	// il riconoscimento della condizione SimulationEnded.
	const stopEpsilon = 1e-9
	if state.Velocity <= 0 {
		state.Velocity = 0
	}
	// Rileva arresto: se velocità <= epsilon OR tempo >= tempo_stop_teorico - epsilon
	reachedStopByVelocity := state.Velocity <= stopEpsilon
	reachedStopByTime := horizontalT >= c.HorizontalStopTime-stopEpsilon
	if reachedStopByVelocity || reachedStopByTime {
		state.Velocity = 0
		state.HorizS = c.HorizontalStopDist
		state.Phase = "stopped"
		state.Completed = true
		state.SimulationEnded = true
		state.SimulationEndTime = inclineTime + c.HorizontalStopTime
		state.SimulationEndHorizS = state.HorizS
		state.Time = state.SimulationEndTime
	}
	return c.withEnergyAndRotation(state)
}

// withEnergyAndRotation arricchisce lo stato cinematico calcolando:
// - velocità angolare e acoelerazione (se corpo rotatorio)
// - angolo di rotazione accumulato
// - energie cinetiche translazionale e rotazionale
// - energia potenziale gravitazionale
// - lavoro totale (differenza di energia cinetica)
func (c InclinedPlaneCalculus) withEnergyAndRotation(state InclinedPlaneSimState) InclinedPlaneSimState {
	// Calcolo dinamica rotazionale solo per corpi rotatori (cilindri, sfere, etc.)
	// ω = v / r, α = a / r, θ_rot = s_totale / r
	if c.ObjectMode == InclinedObjectRotary && c.Radius > 0 {
		state.AngularVelocity = state.Velocity / c.Radius
		state.AngularAcceleration = c.CurrentAngularAcceleration(state.Phase)
		state.RotationAngle = (state.S + state.HorizS) / c.Radius
	} else {
		state.AngularVelocity = 0
		state.AngularAcceleration = 0
		state.RotationAngle = 0
	}

	// Calcolo energetiche:
	// K_trans = (1/2) * m * v^2
	// K_rot = (1/2) * I * ω^2
	// U = m * g * h
	// W_totale = ΔK_trans + ΔK_rot (differenza da stato iniziale)
	state.KineticTranslational = 0.5 * c.Mass * state.Velocity * state.Velocity
	state.KineticRotational = 0.5 * c.MomentOfInertia * state.AngularVelocity * state.AngularVelocity
	state.PotentialEnergy = c.Mass * c.Gravity * state.HBlock
	state.TotalMechanical = state.KineticTranslational + state.KineticRotational + state.PotentialEnergy
	state.TotalWork = (state.KineticTranslational + state.KineticRotational) - (c.InitialKineticTrans + c.InitialKineticRot)
	return state
}

// ComputeInclinedPlaneCalculus è il motore di calcolo principale della fisica del piano inclinato.
// Calcola TUTTI i parametri derivati una sola volta (pre-calculated) da usare poi nelle
// simulazioni. È diviso in blocchi logici:
// 1. Geometria e conversioni angolari
// 2. Calcolo forze base (peso, normale, attrito)
// 3. Logica slides (scivola o no?)
// 4. Dinamica sul piano inclinato (caso block vs rotario)
// 5. Cinematica: tempo e velocità al raggiungimento della base
// 6. Cinematica orizzontale: decelerazione e stop
// 7. Energie iniziali
func ComputeInclinedPlaneCalculus(cfg *config.Config) InclinedPlaneCalculus {
	// BLOCCO 1: Conversioni geometriche e configurazione tipo corpo
	thetaRad := cfg.InclinedTheta * math.Pi / 180.0
	sinTheta := math.Sin(thetaRad)
	cosTheta := math.Cos(thetaRad)
	mg := cfg.InclinedMass * cfg.InclinedGravity

	objectMode := InclinedObjectMode(cfg.InclinedObjectMode)
	if objectMode != InclinedObjectRotary {
		objectMode = InclinedObjectBlock
	}
	rotaryType := InclinedRotaryType(cfg.InclinedRotaryType)
	radius := cfg.InclinedRadius
	if radius < 0 {
		radius = 0
	}

	// BLOCCO 2: Calcolo forze elementari su piano inclinato
	weightParallel := mg * sinTheta
	weightPerp := mg * cosTheta
	normal := weightPerp

	staticFrictionMax := 0.0
	if cfg.InclinedMuSSet {
		staticFrictionMax = cfg.InclinedMuS * normal
	}

	dynamicFriction := 0.0
	netForce := 0.0
	accel := 0.0
	slides := true
	staticFrictionReal := math.Min(weightParallel, staticFrictionMax)

	inertiaFactor := 0.0
	momentOfInertia := 0.0
	isRotary := objectMode == InclinedObjectRotary && radius > 0 && cfg.InclinedMass > 0

	muR := 0.0
	if cfg.InclinedMuRSet {
		muR = cfg.InclinedMuR
		if muR < 0 {
			muR = 0
		}
	}

	// BLOCCO 3: Logica "scivola o no?" e dinamica sul piano inclinato

	// SOTTOBLOCCO 3a: Corpo ROTATORIO (sfera, cilindro, disco)
	if isRotary {
		inertiaFactor = rotaryInertiaFactor(rotaryType)
		momentOfInertia = inertiaFactor * cfg.InclinedMass * radius * radius

		// Sul piano inclinato usa il modello di pura rotolazione, senza dissipazione da attrito volvente.
		// L'attrito di rotolamento μ_r è applicato solo sulla fase orizzontale per frenare.
		dynamicFriction = 0
		availableForce := weightParallel

		if cfg.InclinedInitialVelocity > 0 {
			slides = true
		} else {
			slides = availableForce > 0
		}

		if slides {
			// Accelerazione con rotazione: a = F_net / (m*(1 + I/(m*r²)))
			den := cfg.InclinedMass * (1.0 + inertiaFactor)
			if den > 0 {
				accel = availableForce / den
				if accel < 0 {
					accel = 0
				}
			}
			netForce = cfg.InclinedMass * accel
		}
		staticFrictionMax = 0
		staticFrictionReal = 0
	} else {
		// SOTTOBLOCCO 3b: Corpo BLOCK (obj rigido senza rotazione)
		if cfg.InclinedInitialVelocity > 0 {
			slides = true
		} else if cfg.InclinedMuSSet {
			slides = weightParallel > staticFrictionMax
		}

		if cfg.InclinedMuKSet && (cfg.InclinedMuSSet || cfg.InclinedInitialVelocity > 0) {
			dynamicFriction = cfg.InclinedMuK * normal
		}

		if slides {
			netForce = weightParallel - dynamicFriction
			if cfg.InclinedMass > 0 {
				accel = netForce / cfg.InclinedMass
				if accel < 0 {
					accel = 0
				}
			}
		}
	}

	// BLOCCO 4: Distanza sul piano inclinato (derivata da altezza iniziale)
	distanceToBase := cfg.InclinedLength
	if cfg.InclinedHBlock > 0 && sinTheta > 0 {
		distanceToBase = cfg.InclinedHBlock / sinTheta
	}

	initialHeight := cfg.InclinedLength * sinTheta
	if cfg.InclinedHBlock > 0 {
		initialHeight = cfg.InclinedHBlock
	}

	// BLOCCO 5: Cinematica sul piano inclinato
	// Usa formule di moto uniformemente accelerato: v^2 = v0^2 + 2*a*s, v = v0 + a*t
	v0 := cfg.InclinedInitialVelocity
	if v0 < 0 {
		v0 = 0
	}

	velocityAtBase := v0
	timeToBase := 0.0
	stopsOnIncline := false
	stopDistanceOnIncline := 0.0
	if slides && distanceToBase > 0 {
		// Calcolo velocità al raggiungimento della base usando v^2 = v0^2 + 2*a*s
		v2 := v0*v0 + 2*accel*distanceToBase
		if v2 > 0 {
			velocityAtBase = math.Sqrt(v2)
		}

		// Logica tempo: controlla casi a=0, a>0, a<0
		if math.Abs(accel) < 1e-9 {
			// Accelerazione quasi nulla: moto uniforme
			if v0 > 0 {
				timeToBase = distanceToBase / v0
			}
		} else if accel > 0 {
			// Accelerazione positiva: corpo accelera
			timeToBase = (velocityAtBase - v0) / accel
		} else {
			// Accelerazione negativa: corpo decelera e potrebbe fermarsi sul piano
			// Distanza di arresto con decel: s_stop = v0^2 / (2*|a|)
			stopDistanceOnIncline = (v0 * v0) / (2 * -accel)
			if stopDistanceOnIncline < distanceToBase {
				// Si ferma prima della base
				stopsOnIncline = true
				velocityAtBase = 0
				timeToBase = v0 / -accel
			} else {
				// Raggiunge la base ancora in movimento
				timeToBase = (velocityAtBase - v0) / accel
				if timeToBase < 0 {
					timeToBase = 0
				}
			}
		}
	}

	// BLOCCO 6: Cinematica fase orizzontale (attrito sul terreno piano)
	horizontalDecel := 0.0
	horizontalStopDist := 0.0
	horizontalStopTime := 0.0
	if isRotary {
		// Corpo rotatorio: decelerazione = μ_r * g
		horizontalDecel = muR * cfg.InclinedGravity
		if horizontalDecel > 0 && velocityAtBase > 0 && !stopsOnIncline {
			horizontalStopDist = (velocityAtBase * velocityAtBase) / (2 * horizontalDecel)
			horizontalStopTime = velocityAtBase / horizontalDecel
		}
	} else if cfg.InclinedMuKSet {
		// Block: decelerazione = μₖ * g
		horizontalDecel = cfg.InclinedMuK * cfg.InclinedGravity
		if horizontalDecel > 0 && velocityAtBase > 0 && !stopsOnIncline {
			horizontalStopDist = (velocityAtBase * velocityAtBase) / (2 * horizontalDecel)
			horizontalStopTime = velocityAtBase / horizontalDecel
		}
	}

	// BLOCCO 7: Energie iniziali (per calcolo lavoro dinamico)
	initialOmega := 0.0
	if isRotary && radius > 0 {
		initialOmega = v0 / radius
	}
	initialKTrans := 0.5 * cfg.InclinedMass * v0 * v0
	initialKRot := 0.5 * momentOfInertia * initialOmega * initialOmega
	initialMechanical := initialKTrans + initialKRot + cfg.InclinedMass*cfg.InclinedGravity*initialHeight

	return InclinedPlaneCalculus{
		Theta:                  cfg.InclinedTheta,
		InitialHeight:          initialHeight,
		ObjectMode:             objectMode,
		RotaryType:             rotaryType,
		Radius:                 radius,
		MuR:                    muR,
		WeightParallel:         weightParallel,
		WeightPerp:             weightPerp,
		Normal:                 normal,
		StaticFrictionMax:      staticFrictionMax,
		StaticFrictionReal:     staticFrictionReal,
		DynamicFriction:        dynamicFriction,
		NetForce:               netForce,
		Acceleration:           accel,
		Slides:                 slides,
		RotaryInertiaFactor:    inertiaFactor,
		MomentOfInertia:        momentOfInertia,
		DistanceToBase:         distanceToBase,
		VelocityAtBase:         velocityAtBase,
		TimeToBase:             timeToBase,
		HorizontalDecel:        horizontalDecel,
		HorizontalStopDist:     horizontalStopDist,
		HorizontalStopTime:     horizontalStopTime,
		InitialVelocity:        v0,
		StopsOnIncline:         stopsOnIncline,
		StopDistanceOnIncline:  stopDistanceOnIncline,
		Mass:                   cfg.InclinedMass,
		Gravity:                cfg.InclinedGravity,
		InitialKineticTrans:    initialKTrans,
		InitialKineticRot:      initialKRot,
		InitialMechanicalTotal: initialMechanical,
	}
}

// UpdateInclinedPlaneCalculus ricalcola il modello fisico e sincronizza i risultati nella configurazione globale per la UI. Usato quando l'utente modifica parametri.
func UpdateInclinedPlaneCalculus() error {
	calc := ComputeInclinedPlaneCalculus(config.GlobalConfig)
	return config.UpdateConfig(func(cfg *config.Config) {
		cfg.InclinedWeightParallel = calc.WeightParallel
		cfg.InclinedWeightPerp = calc.WeightPerp
		cfg.InclinedNormal = calc.Normal
		cfg.InclinedStaticFriction = calc.StaticFrictionReal
		cfg.InclinedDynamicFriction = calc.DynamicFriction
		cfg.InclinedNetForce = calc.NetForce
		cfg.InclinedAcceleration = calc.Acceleration
		cfg.InclinedSlides = calc.Slides
	})
}

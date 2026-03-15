package scenes

import (
	"math"
	"physiGo/config"
)

// InclinedPlaneCalculus holds all pre-computed physics values for the inclined plane.
type InclinedPlaneCalculus struct {
	// Input geometry
	Theta         float64
	InitialHeight float64

	// Forces
	WeightParallel     float64
	WeightPerp         float64
	Normal             float64
	StaticFrictionMax  float64
	StaticFrictionReal float64
	DynamicFriction    float64
	NetForce           float64
	Acceleration       float64
	Slides             bool

	// Kinematics on incline
	DistanceToBase        float64
	VelocityAtBase        float64
	TimeToBase            float64
	InitialVelocity       float64
	StopsOnIncline        bool
	StopDistanceOnIncline float64

	// Kinematics on ground after base
	HorizontalDecel    float64
	HorizontalStopDist float64
	HorizontalStopTime float64
}

// InclinedPlaneSimState is the full kinematic state of the simulation at a given instant.
type InclinedPlaneSimState struct {
	Time     float64
	S        float64 // distance traveled along the incline
	HorizS   float64 // distance traveled on the horizontal ground
	Velocity float64
	HBlock   float64 // current block height
	Phase    string  // "ready", "incline", "horizontal", "stopped"

	BaseReached       bool
	BaseReachTime     float64
	BaseReachVelocity float64
	BaseReachDistance float64

	SimulationEnded     bool
	SimulationEndTime   float64
	SimulationEndHorizS float64

	Completed bool
}

// TotalDuration returns the total simulated time for the full motion (incline + optional ground stop).
// Returns 0 when the block does not slide.
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

// SimProgress returns the simulation progress in [0,1] for the given time.
func (c InclinedPlaneCalculus) SimProgress(t float64) float64 {
	total := c.TotalDuration()
	if total <= 0 {
		return 0
	}
	p := t / total
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// BaseReachProgressFraction returns the fraction of total time at which the block reaches the base,
// or -1 if not applicable.
func (c InclinedPlaneCalculus) BaseReachProgressFraction() float64 {
	if !c.Slides || c.StopsOnIncline || c.TimeToBase <= 0 {
		return -1
	}
	total := c.TotalDuration()
	if total <= 0 {
		return -1
	}
	p := c.TimeToBase / total
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// CurrentAcceleration returns the acceleration for the given simulation phase.
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

// DistanceFromBase returns the remaining distance to the incline base for the given simulation state.
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

// ComputeStateAtTime computes the full kinematic state at time t, purely from the pre-calculated values.
func (c InclinedPlaneCalculus) ComputeStateAtTime(t float64) InclinedPlaneSimState {
	h0 := c.InitialHeight

	// Block does not slide: always at rest.
	if !c.Slides {
		return InclinedPlaneSimState{
			Time:     0,
			Velocity: c.InitialVelocity,
			HBlock:   h0,
			Phase:    "ready",
		}
	}

	if t < 0 {
		t = 0
	}

	// Clamp to end when a finite stop exists.
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
		return state
	}

	inclineTime := c.TimeToBase

	// — Block stops on the incline —
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
		return state
	}

	// — Still on the incline —
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
		return state
	}

	// — On the horizontal ground —
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

	if c.HorizontalDecel <= 0 {
		state.HorizS = c.VelocityAtBase * horizontalT
		state.Velocity = c.VelocityAtBase
		return state
	}

	vBase := c.VelocityAtBase
	decel := c.HorizontalDecel
	state.HorizS = vBase*horizontalT - 0.5*decel*horizontalT*horizontalT
	state.Velocity = vBase - decel*horizontalT

	if state.Velocity <= 0 || horizontalT >= c.HorizontalStopTime {
		state.Velocity = 0
		state.HorizS = c.HorizontalStopDist
		state.Phase = "stopped"
		state.Completed = true
		state.SimulationEnded = true
		state.SimulationEndTime = inclineTime + c.HorizontalStopTime
		state.SimulationEndHorizS = state.HorizS
		state.Time = state.SimulationEndTime
	}
	return state
}

func ComputeInclinedPlaneCalculus(cfg *config.Config) InclinedPlaneCalculus {
	thetaRad := cfg.InclinedTheta * math.Pi / 180.0
	sinTheta := math.Sin(thetaRad)
	cosTheta := math.Cos(thetaRad)
	mg := cfg.InclinedMass * cfg.InclinedGravity

	weightParallel := mg * sinTheta
	weightPerp := mg * cosTheta
	normal := weightPerp

	staticFrictionMax := 0.0
	if cfg.InclinedMuSSet {
		staticFrictionMax = cfg.InclinedMuS * normal
	}
	staticFrictionReal := math.Min(weightParallel, staticFrictionMax)

	// The block starts moving only when the tangential component exceeds max static friction.
	slides := true
	if cfg.InclinedInitialVelocity > 0 {
		slides = true
	} else if cfg.InclinedMuSSet {
		slides = weightParallel > staticFrictionMax
	}

	// Dynamic friction on incline is active when mu_k is set and either static friction is active
	// or an initial velocity is provided (forced motion case).
	dynamicFriction := 0.0
	if cfg.InclinedMuKSet && (cfg.InclinedMuSSet || cfg.InclinedInitialVelocity > 0) {
		dynamicFriction = cfg.InclinedMuK * normal
	}

	netForce := 0.0
	accel := 0.0
	if slides {
		netForce = weightParallel - dynamicFriction
		if cfg.InclinedMass > 0 {
			accel = netForce / cfg.InclinedMass
			if accel < 0 {
				accel = 0
			}
		}
	}

	distanceToBase := cfg.InclinedLength
	if cfg.InclinedHBlock > 0 && sinTheta > 0 {
		distanceToBase = cfg.InclinedHBlock / sinTheta
	}

	initialHeight := cfg.InclinedLength * sinTheta
	if cfg.InclinedHBlock > 0 {
		initialHeight = cfg.InclinedHBlock
	}

	v0 := cfg.InclinedInitialVelocity
	if v0 < 0 {
		v0 = 0
	}

	velocityAtBase := v0
	timeToBase := 0.0
	stopsOnIncline := false
	stopDistanceOnIncline := 0.0
	if slides && distanceToBase > 0 {
		v2 := v0*v0 + 2*accel*distanceToBase
		if v2 > 0 {
			velocityAtBase = math.Sqrt(v2)
		}

		if math.Abs(accel) < 1e-9 {
			if v0 > 0 {
				timeToBase = distanceToBase / v0
			}
		} else if accel > 0 {
			timeToBase = (velocityAtBase - v0) / accel
		} else {
			// Decelerating on incline: may stop before reaching the base.
			stopDistanceOnIncline = (v0 * v0) / (2 * -accel)
			if stopDistanceOnIncline < distanceToBase {
				stopsOnIncline = true
				velocityAtBase = 0
				timeToBase = v0 / -accel
			} else {
				timeToBase = (velocityAtBase - v0) / accel
				if timeToBase < 0 {
					timeToBase = 0
				}
			}
		}
	}

	horizontalDecel := 0.0
	horizontalStopDist := 0.0
	horizontalStopTime := 0.0
	if cfg.InclinedMuKSet {
		horizontalDecel = cfg.InclinedMuK * cfg.InclinedGravity
		if horizontalDecel > 0 && velocityAtBase > 0 && !stopsOnIncline {
			horizontalStopDist = (velocityAtBase * velocityAtBase) / (2 * horizontalDecel)
			horizontalStopTime = velocityAtBase / horizontalDecel
		}
	}

	return InclinedPlaneCalculus{
		Theta:                 cfg.InclinedTheta,
		InitialHeight:         initialHeight,
		WeightParallel:        weightParallel,
		WeightPerp:            weightPerp,
		Normal:                normal,
		StaticFrictionMax:     staticFrictionMax,
		StaticFrictionReal:    staticFrictionReal,
		DynamicFriction:       dynamicFriction,
		NetForce:              netForce,
		Acceleration:          accel,
		Slides:                slides,
		DistanceToBase:        distanceToBase,
		VelocityAtBase:        velocityAtBase,
		TimeToBase:            timeToBase,
		HorizontalDecel:       horizontalDecel,
		HorizontalStopDist:    horizontalStopDist,
		HorizontalStopTime:    horizontalStopTime,
		InitialVelocity:       v0,
		StopsOnIncline:        stopsOnIncline,
		StopDistanceOnIncline: stopDistanceOnIncline,
	}
}

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

package scenes

import (
	"math"
	"physiGo/config"
)

type InclinedPlaneCalculus struct {
	WeightParallel        float64
	WeightPerp            float64
	Normal                float64
	StaticFrictionMax     float64
	StaticFrictionReal    float64
	DynamicFriction       float64
	NetForce              float64
	Acceleration          float64
	Slides                bool
	DistanceToBase        float64
	VelocityAtBase        float64
	TimeToBase            float64
	HorizontalDecel       float64
	HorizontalStopDist    float64
	HorizontalStopTime    float64
	InitialVelocity       float64
	StopsOnIncline        bool
	StopDistanceOnIncline float64
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

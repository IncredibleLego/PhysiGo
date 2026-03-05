package scenes

import (
	"math"
	"physiGo/config"
)

type InclinedPlaneCalculus struct {
	WeightParallel  float64
	WeightPerp      float64
	Normal          float64
	StaticFriction  float64
	DynamicFriction float64
	NetForce        float64
	Acceleration    float64
	Slides          bool
}

func ComputeInclinedPlaneCalculus(cfg *config.Config) InclinedPlaneCalculus {
	thetaRad := cfg.InclinedTheta * math.Pi / 180.0
	mg := cfg.InclinedMass * cfg.InclinedGravity

	weightParallel := mg * math.Sin(thetaRad)
	weightPerp := mg * math.Cos(thetaRad)
	normal := weightPerp

	staticFriction := normal
	if cfg.InclinedMuSSet {
		staticFriction = cfg.InclinedMuS * normal
	}

	slides := weightParallel <= staticFriction

	dynamicFriction := normal
	if cfg.InclinedMuKSet {
		dynamicFriction = cfg.InclinedMuK * normal
	}

	netForce := 0.0
	accel := 0.0
	if slides {
		netForce = weightParallel - dynamicFriction
		if cfg.InclinedMuKSet {
			accel = cfg.InclinedGravity * (math.Sin(thetaRad) - cfg.InclinedMuK*math.Cos(thetaRad))
		} else {
			accel = cfg.InclinedGravity * (math.Sin(thetaRad) - math.Cos(thetaRad))
		}
	}

	return InclinedPlaneCalculus{
		WeightParallel:  weightParallel,
		WeightPerp:      weightPerp,
		Normal:          normal,
		StaticFriction:  staticFriction,
		DynamicFriction: dynamicFriction,
		NetForce:        netForce,
		Acceleration:    accel,
		Slides:          slides,
	}
}

func UpdateInclinedPlaneCalculus() error {
	calc := ComputeInclinedPlaneCalculus(config.GlobalConfig)
	return config.UpdateConfig(func(cfg *config.Config) {
		cfg.InclinedWeightParallel = calc.WeightParallel
		cfg.InclinedWeightPerp = calc.WeightPerp
		cfg.InclinedNormal = calc.Normal
		cfg.InclinedStaticFriction = calc.StaticFriction
		cfg.InclinedDynamicFriction = calc.DynamicFriction
		cfg.InclinedNetForce = calc.NetForce
		cfg.InclinedAcceleration = calc.Acceleration
		cfg.InclinedSlides = calc.Slides
	})
}

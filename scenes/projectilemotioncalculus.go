package scenes

import (
	"errors"
	"fmt"
	"math"
	"physiGo/config"
)

const projectileEpsilon = 1e-6

type ProjectileMotionCalculus struct {
	V0         float64
	ThetaDeg   float64
	ThetaRad   float64
	H0         float64
	Range      float64
	FlightTime float64
	G          float64

	Vx0 float64
	Vy0 float64

	ApexTime float64
	ApexX    float64
	ApexY    float64
}

type ProjectileMotionSimState struct {
	Time      float64
	X         float64
	Y         float64
	Vx        float64
	Vy        float64
	Speed     float64
	AngleDeg  float64
	Completed bool
}

func (c ProjectileMotionCalculus) TotalDuration() float64 {
	if c.FlightTime <= 0 {
		return 0
	}
	return c.FlightTime
}

func (c ProjectileMotionCalculus) SimProgress(t float64) float64 {
	total := c.TotalDuration()
	if total <= 0 {
		return 0
	}
	return clamp01(t / total)
}

func (c ProjectileMotionCalculus) ComputeStateAtTime(t float64) ProjectileMotionSimState {
	if t < 0 {
		t = 0
	}
	if c.FlightTime > 0 && t > c.FlightTime {
		t = c.FlightTime
	}

	// Equazioni parametriche del moto del proiettile:
	// x(t) = vx0*t
	// y(t) = h0 + vy0*t - (1/2)g t^2
	x := c.Vx0 * t
	y := c.H0 + c.Vy0*t - 0.5*c.G*t*t
	if y < 0 && c.FlightTime > 0 {
		y = 0
	}
	// Componenti della velocita nel tempo:
	// vx(t) = vx0, vy(t) = vy0 - g*t
	vx := c.Vx0
	vy := c.Vy0 - c.G*t
	// Modulo della velocita e angolo istantaneo della velocita.
	speed := math.Hypot(vx, vy)
	angle := math.Atan2(vy, vx) * 180 / math.Pi

	state := ProjectileMotionSimState{
		Time:     t,
		X:        x,
		Y:        y,
		Vx:       vx,
		Vy:       vy,
		Speed:    speed,
		AngleDeg: angle,
	}
	if c.FlightTime > 0 && t >= c.FlightTime {
		// Forza lo stato finale esattamente all'impatto (suolo a distanza R).
		state.Completed = true
		state.X = c.Range
		state.Y = 0
	}
	return state
}

func ComputeProjectileMotionCalculus(cfg *config.Config) (ProjectileMotionCalculus, error) {
	solution, err := SolveProjectileMotion(
		cfg.ProjectileV0,
		cfg.ProjectileTheta,
		cfg.ProjectileH,
		cfg.ProjectileRange,
		cfg.ProjectileTime,
		cfg.ProjectileGravity,
	)
	if err != nil {
		return ProjectileMotionCalculus{}, err
	}

	thetaRad := solution.ThetaDeg * math.Pi / 180.0
	// Scomposizione della velocita iniziale nelle componenti orizzontale e verticale.
	vx0 := solution.V0 * math.Cos(thetaRad)
	vy0 := solution.V0 * math.Sin(thetaRad)

	// L'apice si ha quando vy(t)=0 => t_apice = vy0/g (limitato all'interno del volo).
	apexTime := 0.0
	if solution.G > 0 {
		apexTime = vy0 / solution.G
		if apexTime < 0 {
			apexTime = 0
		}
		if apexTime > solution.FlightTime {
			apexTime = solution.FlightTime
		}
	}
	// Calcolo delle coordinate dell'apice all'istante t_apice.
	apexX := vx0 * apexTime
	apexY := solution.H0 + vy0*apexTime - 0.5*solution.G*apexTime*apexTime
	if apexY < 0 {
		apexY = 0
	}

	return ProjectileMotionCalculus{
		V0:         solution.V0,
		ThetaDeg:   solution.ThetaDeg,
		ThetaRad:   thetaRad,
		H0:         solution.H0,
		Range:      solution.Range,
		FlightTime: solution.FlightTime,
		G:          solution.G,
		Vx0:        vx0,
		Vy0:        vy0,
		ApexTime:   apexTime,
		ApexX:      apexX,
		ApexY:      apexY,
	}, nil
}

type ProjectileMotionSolution struct {
	V0         float64
	ThetaDeg   float64
	H0         float64
	Range      float64
	FlightTime float64
	G          float64
}

func SolveProjectileMotion(v0, thetaDeg, h0, rg, tf, g float64) (ProjectileMotionSolution, error) {
	if g <= 0 {
		return ProjectileMotionSolution{}, errors.New("g must be greater than 0")
	}
	if h0 < 0 {
		return ProjectileMotionSolution{}, errors.New("h must be >= 0")
	}

	// Servono almeno due input non nulli dell'utente (oltre alla gravita)
	// per ottenere una configurazione del moto determinabile.
	knownCount := 0
	if v0 > projectileEpsilon {
		knownCount++
	}
	if thetaDeg > projectileEpsilon {
		knownCount++
	}
	if h0 > projectileEpsilon {
		knownCount++
	}
	if rg > projectileEpsilon {
		knownCount++
	}
	if tf > projectileEpsilon {
		knownCount++
	}
	if knownCount < 2 {
		return ProjectileMotionSolution{}, errors.New("insert at least 2 values among V0, theta, h, R, t")
	}

	knownV0 := v0 > projectileEpsilon
	knownTheta := thetaDeg > projectileEpsilon
	knownH := true
	knownR := rg > projectileEpsilon
	knownT := tf > projectileEpsilon

	if knownTheta && (thetaDeg < 0 || thetaDeg >= 90) {
		return ProjectileMotionSolution{}, errors.New("theta must be in [0, 90)")
	}

	// Chiusura iterativa: ogni ramo prova a ricavare un'incognita da valori noti.
	// Il ciclo termina quando non si riesce piu a inferire nessuna nuova variabile.
	for step := 0; step < 16; step++ {
		changed := false

		// Scorciatoia per lancio orizzontale (theta = 0): con v0 e h noti,
		// ricava il tempo dalla caduta libera e la gittata da x = v0*t.
		if knownV0 && knownH && !knownTheta && !knownT {
			if h0 <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("for v0-only launch, h must be > 0")
			}
			tf = math.Sqrt((2 * h0) / g)
			if tf <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("computed flight time is not valid")
			}
			thetaDeg = 0
			knownTheta = true
			knownT = true
			changed = true
		}

		if knownV0 && knownTheta && !knownT {
			// Da y(tf)=0 con v0, theta e h noti:
			// 0 = h + v0*sin(theta)*t - (1/2)g t^2
			// si prende la radice positiva come tempo di volo fisicamente valido.
			theta := thetaDeg * math.Pi / 180.0
			vy := v0 * math.Sin(theta)
			discriminant := vy*vy + 2*g*h0
			if discriminant < 0 {
				return ProjectileMotionSolution{}, errors.New("invalid inputs: no real flight time")
			}
			tf = (vy + math.Sqrt(discriminant)) / g
			if tf <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("invalid inputs: flight time <= 0")
			}
			knownT = true
			changed = true
		}

		if knownV0 && knownTheta && knownT && !knownR {
			// Spostamento orizzontale all'impatto: R = v0*cos(theta)*t.
			theta := thetaDeg * math.Pi / 180.0
			rg = v0 * math.Cos(theta) * tf
			if rg < -projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("invalid inputs: range < 0")
			}
			if rg < 0 {
				rg = 0
			}
			knownR = true
			changed = true
		}

		if knownR && knownT && knownTheta && !knownV0 {
			// Inversione di R = v0*cos(theta)*t => v0 = R/(cos(theta)*t).
			theta := thetaDeg * math.Pi / 180.0
			den := math.Cos(theta) * tf
			if den <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("invalid inputs for V0 from R, t, theta")
			}
			v0 = rg / den
			if v0 <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("computed V0 is not valid")
			}
			knownV0 = true
			changed = true
		}

		if knownR && knownT && knownV0 && !knownTheta {
			// Inversione di R = v0*cos(theta)*t => cos(theta)=R/(v0*t).
			c := rg / (v0 * tf)
			if c < -1 || c > 1 {
				return ProjectileMotionSolution{}, errors.New("cannot compute theta from V0, R, t")
			}
			if c < -1 {
				c = -1
			}
			if c > 1 {
				c = 1
			}
			thetaDeg = math.Acos(c) * 180 / math.Pi
			if thetaDeg < 0 || thetaDeg >= 90 {
				return ProjectileMotionSolution{}, errors.New("computed theta outside valid range")
			}
			knownTheta = true
			changed = true
		}

		if knownT && knownH && knownTheta && !knownV0 {
			// Inversione di y(tf)=0 per ricavare v0:
			// v0 = ( (1/2)g t^2 - h ) / ( t*sin(theta) ).
			theta := thetaDeg * math.Pi / 180.0
			den := tf * math.Sin(theta)
			if den <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("cannot compute V0 from t, h, theta")
			}
			v0 = (0.5*g*tf*tf - h0) / den
			if v0 <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("computed V0 is not valid")
			}
			knownV0 = true
			changed = true
		}

		if knownT && knownH && knownV0 && !knownTheta {
			// Inversione di y(tf)=0 per ricavare theta tramite sin(theta):
			// sin(theta) = ( (1/2)g t^2 - h ) / (v0*t).
			s := (0.5*g*tf*tf - h0) / (v0 * tf)
			if s < -1 || s > 1 {
				return ProjectileMotionSolution{}, errors.New("cannot compute theta from V0, h, t")
			}
			if s < -1 {
				s = -1
			}
			if s > 1 {
				s = 1
			}
			thetaDeg = math.Asin(s) * 180 / math.Pi
			if thetaDeg < 0 || thetaDeg >= 90 {
				return ProjectileMotionSolution{}, errors.New("computed theta outside valid range")
			}
			knownTheta = true
			changed = true
		}

		if knownR && knownH && knownTheta && !knownV0 {
			// Formula chiusa dall'equazione della traiettoria con R, h, theta noti:
			// v0^2 = g*R^2 / ( 2*cos^2(theta)*(R*tan(theta)-h) ).
			theta := thetaDeg * math.Pi / 180.0
			den := 2 * math.Cos(theta) * math.Cos(theta) * (rg*math.Tan(theta) - h0)
			if den <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("cannot compute V0 from R, h, theta")
			}
			v0sq := g * rg * rg / den
			if v0sq <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("computed V0^2 is not valid")
			}
			v0 = math.Sqrt(v0sq)
			knownV0 = true
			changed = true
		}

		if knownV0 && knownH && knownR && !knownTheta {
			// Risolve la quadratica in u = tan(theta) dalla traiettoria in x=R:
			// h + R*u - (g*R^2/(2*v0^2))*(1+u^2) = 0.
			// Possono uscire due angoli validi; viene scelto quello piu piccolo.
			a := g * rg * rg / (2 * v0 * v0)
			if a <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("cannot compute theta from V0, h, R")
			}
			disc := rg*rg - 4*a*(a-h0)
			if disc < 0 {
				return ProjectileMotionSolution{}, errors.New("inputs are inconsistent: no real theta")
			}
			sqrtDisc := math.Sqrt(disc)
			u1 := (rg + sqrtDisc) / (2 * a)
			u2 := (rg - sqrtDisc) / (2 * a)
			cand := []float64{u1, u2}
			best := 0.0
			for _, u := range cand {
				if u <= projectileEpsilon {
					continue
				}
				th := math.Atan(u) * 180 / math.Pi
				if th < 0 || th >= 90 {
					continue
				}
				if best == 0 || th < best {
					best = th
				}
			}
			if best == 0 {
				return ProjectileMotionSolution{}, errors.New("computed theta outside valid range")
			}
			thetaDeg = best
			knownTheta = true
			changed = true
		}

		if knownR && knownT && knownH && !knownV0 && !knownTheta {
			// Ricostruzione diretta delle componenti:
			// vx = R/t, vy = ((1/2)g t^2 - h)/t,
			// then v0 = sqrt(vx^2+vy^2), theta = atan2(vy,vx).
			vx := rg / tf
			vy := (0.5*g*tf*tf - h0) / tf
			v0 = math.Hypot(vx, vy)
			if v0 <= projectileEpsilon {
				return ProjectileMotionSolution{}, errors.New("computed V0 is not valid")
			}
			thetaDeg = math.Atan2(vy, vx) * 180 / math.Pi
			if thetaDeg < 0 || thetaDeg >= 90 {
				return ProjectileMotionSolution{}, errors.New("computed theta outside valid range")
			}
			knownV0 = true
			knownTheta = true
			changed = true
		}

		if !changed {
			break
		}
	}

	if !(knownV0 && knownTheta && knownT && knownR) {
		return ProjectileMotionSolution{}, errors.New("insufficient/ambiguous inputs to solve all variables")
	}

	// Controllo finale di consistenza: ricalcola le equazioni d'impatto
	// usando i valori risolti.
	theta := thetaDeg * math.Pi / 180.0
	calcR := v0 * math.Cos(theta) * tf
	calcY := h0 + v0*math.Sin(theta)*tf - 0.5*g*tf*tf

	rTol := 1e-3 * math.Max(1, rg)
	if math.Abs(calcR-rg) > rTol {
		return ProjectileMotionSolution{}, fmt.Errorf("input mismatch on R (expected %.3f, got %.3f)", calcR, rg)
	}
	if math.Abs(calcY) > 1e-2 {
		return ProjectileMotionSolution{}, errors.New("inputs are inconsistent with y(t_f)=0")
	}

	if rg < 0 {
		rg = 0
	}
	if tf < 0 {
		tf = 0
	}

	return ProjectileMotionSolution{
		V0:         v0,
		ThetaDeg:   thetaDeg,
		H0:         h0,
		Range:      rg,
		FlightTime: tf,
		G:          g,
	}, nil
}

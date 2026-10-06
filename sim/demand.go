package sim

// Zone indexes for per-zone arrays (Tax, Demand).
const (
	R = iota
	C
	I
)

var zoneNames = [3]string{"residential", "commercial", "industrial"}

// RawDemand computes the target demand for each zone from aggregates,
// before tax and smoothing (SPEC §6.2).
func RawDemand(s Stats) [3]float64 {
	p := float64(s.Residents)
	l := s.Workforce()
	jobs := float64(s.CommJobs + s.IndJobs)
	return [3]float64{
		clamp((jobs+resBootstrap-l)/max(jobs+resBootstrap, 1), -1, 1),
		clamp((comPerResident*p-float64(s.CommJobs))/max(comPerResident*p, comMinBase), -1, 1),
		clamp((indPerWorker*l+indBootstrap-float64(s.IndJobs))/max(indPerWorker*l+indBootstrap, 1), -1, 1),
	}
}

// TaxMod is the demand penalty for a tax rate (negative below neutral).
func TaxMod(rate int) float64 { return taxSlope * float64(rate-taxNeutral) }

func (c *City) updateDemand() {
	raw := RawDemand(c.Stats)
	if c.has(Stadium) {
		raw[R] += stadiumPull
	}
	for z, b := range c.demandBoost() {
		raw[z] += b
	}
	for z := range c.Demand {
		target := clamp(raw[z]-TaxMod(c.Tax[z]), -1, 1)
		c.Demand[z] += demandSmooth * (target - c.Demand[z])

		if c.Demand[z] > demandHigh {
			c.highDays[z]++
			if c.highDays[z] == demandHighDays {
				c.Logf(Info, "%s demand high", zoneNames[z])
			}
		} else {
			c.highDays[z] = 0
		}
	}
}

// stadiumPull is extra residential demand while a stadium stands.
const stadiumPull = 0.1

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

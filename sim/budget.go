package sim

import "fmt"

// Budget constants (SPEC §6.7).
const (
	incomeScale    = 0.1
	LoanAmount     = 10000
	LoanPayment    = 450
	LoanMonths     = 24
	TaxMax         = 20
	debtWarnMonths = 9
	bankruptMonths = 12
)

// Upkeep per month.
var (
	upkeepRoad = 0.5
	upkeepLine = 0.25
	upkeepPipe = 0.25
)

const cityHallTaxBonus = 1.05

// has reports whether at least one building of kind k stands.
func (c *City) has(k Kind) bool {
	for i := range c.Tiles {
		if c.Tiles[i].Kind == k {
			return true
		}
	}
	return false
}

// Loan is an outstanding loan.
type Loan struct {
	Remaining  float64
	MonthsLeft int
}

// Ledger is one month's money, for the budget panel.
type Ledger struct {
	Income    [3]float64 // R, C, I taxes
	Roads     float64
	Lines     float64
	Pipes     float64
	Buildings map[Kind]float64
	LoanPaid  float64
	TotalIn   float64
	TotalOut  float64
	Net       float64
}

// Forecast computes this month's ledger from the city as it is now.
func (c *City) Forecast() Ledger {
	l := Ledger{Buildings: map[Kind]float64{}}
	s := c.Stats
	bonus := 1.0
	if c.has(CityHall) {
		bonus = cityHallTaxBonus
	}
	l.Income = [3]float64{
		bonus * incomeScale * float64(s.Residents*c.Tax[R]),
		bonus * incomeScale * float64(s.CommJobs*c.Tax[C]),
		bonus * incomeScale * float64(s.IndJobs*c.Tax[I]),
	}
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if t.Kind == Road {
			l.Roads += upkeepRoad
		}
		if t.Line {
			l.Lines += upkeepLine
		}
		if t.Pipe {
			l.Pipes += upkeepPipe
		}
		if t.IsBuilding() && int(t.Anchor) == i {
			l.Buildings[t.Kind] += spec(t.Kind).Upkeep
		}
	}
	if c.Loan != nil {
		l.LoanPaid = min(LoanPayment, c.Loan.Remaining)
	}
	l.TotalIn = l.Income[0] + l.Income[1] + l.Income[2]
	l.TotalOut = l.Roads + l.Lines + l.Pipes + l.LoanPaid
	for _, v := range l.Buildings {
		l.TotalOut += v
	}
	l.Net = l.TotalIn - l.TotalOut
	return l
}

// monthly settles the books at the start of each month.
func (c *City) monthly() {
	l := c.Forecast()
	c.Funds += l.Net
	c.record()
	if c.Loan != nil {
		c.Loan.Remaining -= l.LoanPaid
		c.Loan.MonthsLeft--
		if c.Loan.Remaining <= 0.5 {
			c.Loan = nil
			c.Logf(Info, "loan paid off")
		}
	}
	// One line a year, plus a warning when the books tip into the red.
	prevNet := c.LastMonth.Net
	c.LastMonth = l
	c.yearNet += l.Net
	if l.Net < 0 && (prevNet >= 0 || c.Day == DaysPerMonth) {
		c.Logf(Warn, "budget: spending $%.0f more than taxes bring in each month", -l.Net)
	}
	if c.Day%(DaysPerMonth*MonthsPerYear) == 0 {
		lvl := Info
		if c.yearNet < 0 {
			lvl = Warn
		}
		c.Logf(lvl, "year %d closed: net %+.0f, funds $%.0f", c.Day/(DaysPerMonth*MonthsPerYear), c.yearNet, c.Funds)
		c.yearNet = 0
	}

	if c.Funds >= 0 {
		if c.DebtMonths > 0 {
			c.Logf(Info, "out of debt")
		}
		c.DebtMonths = 0
		return
	}
	c.DebtMonths++
	c.EverInDebt = true
	switch {
	case c.DebtMonths == 1:
		c.Logf(Err, "in debt: building is blocked until funds are positive")
	case c.DebtMonths == debtWarnMonths:
		c.Logf(Err, "%d months in debt: bankruptcy in %d months", debtWarnMonths, bankruptMonths-debtWarnMonths)
	case c.DebtMonths >= bankruptMonths:
		c.Bankrupt = true
		c.Logf(Err, "bankrupt after %d months in debt", bankruptMonths)
	}
}

// SetTax sets the tax rate for zone z (R, C, I), or all zones if z < 0.
func (c *City) SetTax(z, rate int) error {
	if rate < 0 || rate > TaxMax {
		return fmt.Errorf("tax must be 0..%d", TaxMax)
	}
	for i := range c.Tax {
		if z < 0 || z == i {
			c.Tax[i] = rate
		}
	}
	return nil
}

// TakeLoan borrows LoanAmount, one loan at a time.
func (c *City) TakeLoan() error {
	if c.Loan != nil {
		return fmt.Errorf("already repaying a loan ($%.0f left)", c.Loan.Remaining)
	}
	if c.Bankrupt {
		return fmt.Errorf("no bank lends to a bankrupt city")
	}
	c.Funds += LoanAmount
	c.Loan = &Loan{Remaining: LoanPayment * LoanMonths, MonthsLeft: LoanMonths}
	c.Logf(Info, "loan: +$%d, repaying $%d/month for %d months", LoanAmount, LoanPayment, LoanMonths)
	return nil
}

// Repay pays off the loan early.
func (c *City) Repay() error {
	switch {
	case c.Loan == nil:
		return fmt.Errorf("no loan to repay")
	case c.Funds < c.Loan.Remaining:
		return fmt.Errorf("need $%.0f to repay", c.Loan.Remaining)
	}
	c.Funds -= c.Loan.Remaining
	c.Logf(Info, "loan repaid early ($%.0f)", c.Loan.Remaining)
	c.Loan = nil
	return nil
}

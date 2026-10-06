package sim

import "testing"

func TestForecast(t *testing.T) {
	c := flat(10, 10)
	c.Apply(ToolRoad, RectPts(Pt{0, 0}, Pt{9, 0}), false)
	c.Apply(ToolLine, RectPts(Pt{0, 1}, Pt{3, 1}), false)
	c.Apply(ToolPlant, []Pt{{0, 5}}, false)
	c.Stats = Stats{Residents: 1000, CommJobs: 100, IndJobs: 200}
	l := c.Forecast()
	if !near(l.TotalIn, 0.1*(1000*9+100*9+200*9)) {
		t.Errorf("income %v", l.TotalIn)
	}
	if !near(l.TotalOut, 10*0.5+4*0.25+100) {
		t.Errorf("upkeep %v", l.TotalOut)
	}
}

func TestMonthRollover(t *testing.T) {
	c := flat(10, 10)
	c.Apply(ToolPlant, []Pt{{0, 0}}, false)
	f := c.Funds
	for i := 0; i < DaysPerMonth*TicksPerDay; i++ {
		c.Tick()
	}
	if c.Funds != f {
		t.Fatal("charged before the month ended")
	}
	c.Tick()
	if !near(c.Funds, f-100) || c.LastMonth.Net != -100 {
		t.Errorf("funds %v last %+v", c.Funds, c.LastMonth)
	}
}

func TestLoan(t *testing.T) {
	c := flat(4, 4)
	if err := c.TakeLoan(); err != nil || c.Funds != StartingFunds+LoanAmount {
		t.Fatal(err)
	}
	if c.TakeLoan() == nil {
		t.Error("second loan allowed")
	}
	for m := 0; m < LoanMonths; m++ {
		c.monthly()
	}
	if c.Loan != nil || !near(c.Funds, StartingFunds+LoanAmount-LoanPayment*LoanMonths) {
		t.Errorf("after schedule: loan %+v funds %v", c.Loan, c.Funds)
	}
	c.TakeLoan()
	c.Funds = 100
	if c.Repay() == nil {
		t.Error("repaid without the money")
	}
	c.Funds = 1e6
	if c.Repay() != nil || c.Loan != nil {
		t.Error("early repayment")
	}
}

func TestBankruptcy(t *testing.T) {
	c := flat(4, 4)
	c.Funds = -50
	for m := 1; m < bankruptMonths; m++ {
		c.monthly()
		if c.Bankrupt {
			t.Fatalf("bankrupt after %d months", m)
		}
	}
	c.monthly()
	if !c.Bankrupt {
		t.Fatal("not bankrupt after 12 months in debt")
	}
	day := c.Day
	c.Tick()
	if c.Day != day {
		t.Error("bankrupt city keeps ticking")
	}
	// Climbing out resets the counter.
	d := flat(4, 4)
	d.Funds = -50
	d.monthly()
	d.Funds = 10
	d.monthly()
	if d.DebtMonths != 0 {
		t.Error("debt counter not reset")
	}
}

func TestSetTax(t *testing.T) {
	c := flat(4, 4)
	if c.SetTax(-1, 12) != nil || c.Tax != [3]int{12, 12, 12} {
		t.Error("all taxes")
	}
	if c.SetTax(I, 3) != nil || c.Tax[I] != 3 || c.Tax[R] != 12 {
		t.Error("one tax")
	}
	if c.SetTax(R, 21) == nil {
		t.Error("tax above max")
	}
}

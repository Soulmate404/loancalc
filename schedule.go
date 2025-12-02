package loancalc

import (
	"time"

	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

var one = decimal.NewFromInt(1)

func AnnuityPayment(principal Decimal, periods int64, rate Decimal) Decimal {

	base1r := rate.Add(one)
	base1rn := base1r.Pow(decimal.NewFromInt(periods))

	numerator := principal.Mul(base1rn).Mul(rate)
	denominator := base1rn.Sub(one)
	return numerator.Div(denominator)
}

// AnnuitySchedule 生成等额本息计划（注入 Clock/Holiday/Round）
func AnnuitySchedule(loanId int64, principal Decimal, periods int64, product *Product, idGenerator IDGenerator) ([]Schedule, error) {
	schedules := make([]Schedule, 0, periods)
	nextDate := func(t time.Time) time.Time {
		n, _ := NextPeriodDate(t, product.PeriodType, product.RollConvention)
		return n
	}
	t := cfg.Clock.Now()
	g := int64(product.GraceTerm)
	r, err := AnnualToPeriodRate(product.Interest, product.PeriodType, product.DayCountConv)
	if err != nil {
		return nil, err
	}
	pwt := AnnuityPayment(principal, periods-g, r)
	for i := int64(1); i <= periods; i++ {
		t = nextDate(t)
		fees := make([]Fee, len(product.Fees))
		copy(fees, product.Fees)
		id := idGenerator()
		for j := 0; j < len(fees); j++ {
			fees[j].ID = idGenerator()
			fees[j].Status = FeeStatusUnPaid
			fees[j].ScheduleId = id
		}
		interest := principal.Mul(r)
		if i <= g {
			s := NewSchedule(id, loanId, int(i), t, decimal.Zero, interest, fees)
			schedules = append(schedules, *s)
			continue
		}
		p := pwt.Sub(interest)
		principal = principal.Sub(p)
		s := NewSchedule(id, loanId, int(i), t, p, interest, fees)
		schedules = append(schedules, *s)
	}
	return schedules, nil
}

func EqualPrincipalPayment(principal Decimal, periods int64, rate Decimal) Decimal {
	p := principal.Div(decimal.NewFromInt(periods))
	i := principal.Mul(rate)
	return p.Add(i)
}

func EqualPrincipalSchedule(loanId int64, principal Decimal, periods int64, product *Product, idGenerator IDGenerator) ([]Schedule, error) {
	schedules := make([]Schedule, 0, periods)
	nextDate := func(t time.Time) time.Time {
		n, _ := NextPeriodDate(t, product.PeriodType, product.RollConvention)
		return n
	}
	t := cfg.Clock.Now()
	r, err := AnnualToPeriodRate(product.Interest, product.PeriodType, product.DayCountConv)
	if err != nil {
		return nil, err
	}
	g := int64(product.GraceTerm)
	p := principal.Div(decimal.NewFromInt(periods - g))
	for i := int64(1); i <= periods; i++ {
		t = nextDate(t)
		fees := make([]Fee, len(product.Fees))
		copy(fees, product.Fees)
		id := idGenerator()
		for j := 0; j < len(fees); j++ {
			fees[j].ID = idGenerator()
			fees[j].Status = FeeStatusUnPaid
			fees[j].ScheduleId = id
		}
		if i <= g {
			interest := principal.Mul(r)
			s := NewSchedule(id, loanId, int(i), t, decimal.Zero, interest, fees)
			schedules = append(schedules, *s)
			continue
		}
		principal = principal.Sub(p)
		interest := principal.Mul(r)
		s := NewSchedule(id, loanId, int(i), t, p, interest, fees)
		schedules = append(schedules, *s)
	}
	return schedules, nil
}

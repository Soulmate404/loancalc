package loancalc

import (
	"errors"

	"time"

	"github.com/shopspring/decimal"
)

func AnnualToPeriodRate(annual decimal.Decimal, pt PeriodType, conv DayCountConv) (decimal.Decimal, error) {
	// 使用可注入的 Clock 以保证可测性
	t := cfg.Clock.Now()
	var next time.Time
	switch pt {
	case PeriodDay:
		next = t.AddDate(0, 0, 1)
	case PeriodMonth:
		next = t.AddDate(0, 1, 0)
	case PeriodYear:
		next = t.AddDate(1, 0, 0)
	case PeriodBiWeek:
		next = t.AddDate(0, 0, 14)
	default:
		return decimal.Zero, errors.New("unknown period type")
	}
	ratio, err := EffectiveInterestRate(t, next, conv)
	if err != nil {
		return decimal.Zero, err
	}
	return ratio.Mul(annual), nil
}

// -------------------- 30/360 U.S. (Bond Basis) --------------------

// Days360US returns (numerator, denominator=360) under 30/360 U.S. rules:
//   - if d1==31 → d1=30
//   - if d2==31 && d1>=30 → d2=30
func Days360US(start, end time.Time) (int, int) {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()

	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 && d1 >= 30 {
		d2 = 30
	}
	days := (y2-y1)*360 + int(m2-m1)*30 + (d2 - d1)
	return days, 360
}

// -------------------- 30E/360 (Eurobond) --------------------

// Days360E returns (numerator, denominator=360) under 30E/360:
//   - both d1,d2 ==31 → 30 unconditionally
func Days360E(start, end time.Time) (int, int) {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()

	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 {
		d2 = 30
	}
	days := (y2-y1)*360 + int(m2-m1)*30 + (d2 - d1)
	return days, 360
}

// -------------------- Actual/360 (Money Market) --------------------

// DaysAct360 returns actual calendar days / 360
func DaysAct360(start, end time.Time) (int, int) {
	return int(end.Sub(start).Hours() / 24), 360
}

// -------------------- Actual/365 (Fixed) --------------------

// DaysAct365 returns actual days / 365 (ignore leap year)
func DaysAct365(start, end time.Time) (int, int) {
	return int(end.Sub(start).Hours() / 24), 365
}

// -------------------- Actual/Actual (ISDA) --------------------

// DaysActActISDA returns actual days / yearDays(where the day falls)
// If the period spans multiple years, caller should split and sum.
func DaysActActISDA(start, end time.Time) (int, int) {
	days := int(end.Sub(start).Hours() / 24)
	// denominator = year days of the year where 'start' lies
	yearBase := YearDays(start)
	return days, yearBase
}

// -------------------- Actual/365 (AFB, 365.25) --------------------

// DaysAct365AFB returns actual days / 365.25
func DaysAct365AFB(start, end time.Time) (int, int) {
	days := int(end.Sub(start).Hours() / 24)
	return days, 36525 // 调用方用 36525/100
}

func EffectiveInterestRate(start, end time.Time, conv DayCountConv) (decimal.Decimal, error) {
	var d, y int
	switch conv {
	case BONDBASIS:
		d, y = Days360US(start, end)
	case EUROBOND:
		d, y = Days360E(start, end)
	case MONEYMARKET:
		d, y = DaysAct360(start, end)
	case FIXED:
		d, y = DaysAct365(start, end)
	case ISDA:
		d, y = DaysActActISDA(start, end)
	case AFB:
		d, y = DaysAct365AFB(start, end)
	default:
		return decimal.NewFromInt(0), errors.New("unsupported day count")
	}
	day := decimal.NewFromInt(int64(d))
	var year decimal.Decimal
	if y == 36525 {
		year = decimal.NewFromFloat(365.25)
	} else {
		year = decimal.NewFromInt(int64(y))
	}
	return day.Div(year), nil

}

// YearDays returns 366 if t is in a leap year, else 365
func YearDays(t time.Time) int {
	yy := t.Year()
	if yy%4 != 0 {
		return 365
	}
	if yy%100 != 0 {
		return 366
	}
	if yy%400 == 0 {
		return 366
	}
	return 365
}

package loancalc

import (
	"fmt"
	"time"
)

type HolidayFunc func(time.Time) bool

// NextPeriodDate 使用可注入的 HolidayProvider 实现跳期
func NextPeriodDate(last time.Time, period PeriodType, roll RollConvention) (time.Time, error) {
	// 由于内部 NextPeriodDate 需要一个 isHoliday func，这里复用中国实现以共享节假日缓存
	// 对于自定义 HolidayProvider，这里优先使用 provider，再回落内部实现
	if _, ok := cfg.Holiday.(ChinaHolidayProvider); ok {
		return nextPeriodDate(last, period, roll, func(t time.Time) bool { return ChinaHolidayProvider{}.IsHoliday(t) })
	}
	// 自定义 provider
	return nextPeriodDate(last, period, roll, cfg.Holiday.IsHoliday)
}

// NextPeriodDate 核心函数：给定“上一期还款日”“期别单位”“跳期规则”，返回下一期还款日
// period 支持 Day/BiWeek/Month/Year，与前面 DayCount 包共用同一套枚举
func nextPeriodDate(last time.Time, period PeriodType, roll RollConvention, isHoliday HolidayFunc) (time.Time, error) {
	var candidate time.Time
	switch period {
	case PeriodDay:
		candidate = last.AddDate(0, 0, 1)
	case PeriodBiWeek:
		candidate = last.AddDate(0, 0, 14)
	case PeriodMonth:
		candidate = last.AddDate(0, 1, 0)
	case PeriodYear:
		candidate = last.AddDate(1, 0, 0)
	default:
		return last, fmt.Errorf("unknown period type: %s", period)
	}
	return applyRoll(candidate, roll, isHoliday), nil
}
func applyRoll(t time.Time, roll RollConvention, isHoliday func(time.Time) bool) time.Time {
	switch roll {
	case Unadjusted:
		return t
	case Following:
		for isHoliday(t) {
			t = t.AddDate(0, 0, 1)
		}
		return t
	case Preceding:
		for isHoliday(t) {
			t = t.AddDate(0, 0, -1)
		}
		return t
	case ModFollow:
		origMonth := t.Month()
		t2 := t

		for isHoliday(t2) {
			t2 = t2.AddDate(0, 0, 1)
		}
		if t2.Month() != origMonth {
			t2 = t
			for isHoliday(t2) {
				t2 = t2.AddDate(0, 0, -1)
			}
		}
		return t2
	}
	return t
}

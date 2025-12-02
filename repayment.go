package loancalc

import (
	"time"

	"github.com/shopspring/decimal"
)

var idxNotFound = -1
var ONE = decimal.NewFromInt(1)

// NormalRepay 主要用于处理到期的自动扣款等业务，不考虑提前还款的可能性
func NormalRepay(l *LoanExtra, amount decimal.Decimal, generator IDGenerator) (remaining decimal.Decimal, err error) {
	remaining = amount
	if len(l.Schedules) == 0 {
		return remaining, ErrNoScheduleFound
	}
	repayment := NewRepayment(generator(), l.ID)
	now := time.Now()
	firstOverdueIdx := idxNotFound
	currentIdx := 0

	// 1. 定位：第一个逾期期次 & 当期（应还日=今天且未结清）
	for i, s := range l.Schedules {
		if s.Status == SchedulePaid {
			continue
		}
		if s.Overdue && firstOverdueIdx == idxNotFound {
			firstOverdueIdx = i
		}
		if !s.Overdue {
			cmp := CompareDate(s.DueDate, now)
			if cmp != 0 {
				//return remaining, ErrTodayNotDueDate
			}
			currentIdx = i
			break
		}
	}

	// 2. 还罚息（只要有逾期记录就还）
	if l.HasOverdue() {
		for _, od := range l.OverdueRecords {
			remaining = od.TryToPay(remaining)
			if remaining.IsZero() {
				return remaining, ErrInsufficientForPenalty
			}
		}
	}

	// 3. 还分期本金/利息/费用
	start := firstOverdueIdx
	if start == idxNotFound {
		start = currentIdx
	}
	//挂逾期的任务交给每天定时的跑批任务
	for i := start; i <= currentIdx; i++ {
		remaining = l.Schedules[i].TryToPay(remaining)
		if remaining.IsZero() && l.Schedules[i].Status != SchedulePaid {
			return remaining, ErrInsufficientForSchedule
		}
	}
	repayment.AddAmount(amount.Sub(remaining))
	l.AddRepayment(*repayment)
	return remaining, nil
}

// PreRepay 统一入口
func PreRepay(l *LoanExtra, amount decimal.Decimal,
	generator IDGenerator, strategy PrepayStrategy) (remaining decimal.Decimal, err error) {

	remaining = amount
	repayment := NewRepayment(generator(), l.ID)
	if l.HasOverdue() {
		for _, od := range l.OverdueRecords {
			remaining = od.TryToPay(remaining)
			if remaining.Equal(decimal.Zero) {
				return remaining, ErrInsufficientForPenalty
			}
		}
	}
	firstOverdueIdx := idxNotFound
	currentIdx := 0
	for i, s := range l.Schedules {
		if s.Status == SchedulePaid {
			continue
		}
		if s.Overdue && firstOverdueIdx == idxNotFound {
			firstOverdueIdx = i
		}
		if !s.Overdue {
			currentIdx = i
			break
		}
	}

	//还逾期账单
	start := firstOverdueIdx
	if start == idxNotFound {
		start = currentIdx
	}
	for i := start; i < currentIdx; i++ {
		remaining = l.Schedules[i].TryToPay(remaining)
		if remaining.IsZero() && l.Schedules[i].Status != SchedulePaid {
			return remaining, ErrInsufficientForSchedule
		}
	}

	/* ========== 4. 提前还款 ========== */
	if remaining.Cmp(decimal.Zero) > 0 {
		remaining, err = prepayCore(l, remaining, generator, strategy)
		if err != nil {
			return remaining, err
		}
	}
	repayment.AddAmount(amount.Sub(remaining))
	l.AddRepayment(*repayment)
	return remaining, nil
}

// 真正的提前还款内核
func prepayCore(l *LoanExtra, money decimal.Decimal,
	gen IDGenerator, strategy PrepayStrategy) (decimal.Decimal, error) {

	outstandingPrincipal := l.OutstandingPrincipal()
	if money.Cmp(outstandingPrincipal.Mul(ONE.Add(l.Product.DefaultRate))) >= 0 {
		// 一次性结清
		for i := 0; i < len(l.Schedules); i++ {
			l.Schedules[i].Status = SchedulePaid
		}
		return money.Sub(outstandingPrincipal), nil
	}

	if strategy == PrepayTermReduction {
		return prepayTermReduction(l, money, gen)
	}
	return prepayPaymentReduction(l, money, gen)
}

/* 缩期：从最后一期往前冲本金，整期抹掉 */
func prepayTermReduction(l *LoanExtra, money decimal.Decimal, gen IDGenerator) (decimal.Decimal, error) {
	for i := len(l.Schedules) - 1; i >= 0; i-- {
		s := &l.Schedules[i]
		if s.Status == SchedulePaid || s.Status == ScheduleRemoved {
			continue
		}
		f := s.Principal.Mul(ONE.Add(l.Product.DefaultRate))
		if money.Cmp(f) >= 0 {
			money = money.Sub(f)
			s.Status = SchedulePaid
		} else {
			newS := NewSchedule(gen(), s.LoanID, s.Period, s.DueDate, s.Principal, s.Interest, s.ServiceFee)
			r := ONE.Add(l.Product.DefaultRate)
			x := money.Div(r)
			newS.Principal = newS.Principal.Sub(x)
			newS.Interest = newS.Principal.Mul(l.Product.Interest)
			l.AddSchedule(*newS)
			s.Status = ScheduleRemoved
			break
		}
	}
	return money, nil
}

/* 减额：保持期数，重新生成等额本息/等额本金计划 */
func prepayPaymentReduction(l *LoanExtra, money decimal.Decimal, gen IDGenerator) (decimal.Decimal, error) {
	newPrincipal := l.OutstandingPrincipal().Sub(money)
	periods := int64(l.OutstandingPeriods())

	var newSchedules []Schedule
	switch l.Product.RepayType {
	case RepayTypeEqualInstallment:
		newSchedules, _ = AnnuitySchedule(l.ID, newPrincipal, periods, l.Product, gen)
	case RepayTypeEqualPrincipal:
		newSchedules, _ = EqualPrincipalSchedule(l.ID, newPrincipal, periods, l.Product, gen)
	default:
		return money, ErrUnSupportRepayType
	}
	// 把旧计划全部标记删除
	for i := 0; i < len(l.Schedules); i++ {
		s := &l.Schedules[i]
		if s.Status != SchedulePaid {
			s.Status = ScheduleRemoved
		}
	}
	// 追加新计划
	for _, ns := range newSchedules {
		l.AddSchedule(ns)
	}
	return decimal.Zero, nil
}
func CompareDate(t1, t2 time.Time) int {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()

	switch {
	case y1 < y2 || (y1 == y2 && m1 < m2) || (y1 == y2 && m1 == m2 && d1 < d2):
		return -1
	case y1 == y2 && m1 == m2 && d1 == d2:
		return 0
	default:
		return 1
	}
}

package loancalc

import (
	"errors"

	"time"

	"github.com/shopspring/decimal"
)

// Fee 费用定义（服务费、管理费、咨询费等）,为了节约结构转换时的开销，计划和产品公用相同的结构体，当作为产品下字段时statues字段为MODULE
type Fee struct {
	ID         int64           `db:"id"`
	ScheduleId int64           `db:"schedule_id"`
	Name       string          `db:"name"` // 费用名称
	Rate       decimal.Decimal `db:"rate"` // 相对本金比例（可为 0）
	Fix        decimal.Decimal `db:"fix"`  // 固定金额（可为 0）
	Status     FeeStatus       `db:"status"`
}

// TODO:将这里处理的更加语义化

func (f *Fee) GetFee(p decimal.Decimal) decimal.Decimal {
	total := decimal.Zero
	if !f.Fix.IsZero() {
		total = f.Fix
	}
	total = total.Add(p.Mul(f.Rate))
	return total
}

// LoanExtra 聚合根：包含计划、费用、提前还款、逾期记录
type LoanExtra struct {
	Loan
	Schedules      []Schedule      `db:"schedules"`       // 生成的计划表
	Repayments     []Repayment     `db:"repayments"`      // 已发生的还款事件
	OverdueRecords []OverdueRecord `db:"overdue_records"` // 逾期记录

}
type Loan struct {
	ID           int64           `db:"id"`
	UserID       int64           `db:"user_id"`
	Principal    decimal.Decimal `db:"principal"`     // 合同本金
	TotalPeriods int             `db:"total_periods"` // 总期数
	Product      *Product        `db:"product"`       //对应的金融产品
	CreatedAt    time.Time       `db:"created_at"`
	Statue       LoanStatus      `db:"statue"`
}

// OverdueRecord 逾期记录（值对象）

// NewLoan 工厂，做一些基本校验
func NewLoan(userID int64, principal decimal.Decimal, totalPeriods int, product *Product) (*Loan, error) {
	if principal.Cmp(decimal.Zero) <= 0 || totalPeriods <= 0 || product == nil {
		return nil, errors.New("invalid params")
	}
	newLoan := &Loan{
		ID:           cfg.IDGenerator(),
		UserID:       userID,
		Principal:    principal,
		TotalPeriods: totalPeriods,
		Product:      product,
		CreatedAt:    time.Now(),
		Statue:       LoanPending,
	}
	return newLoan, nil
}

func (l *Loan) ToLoanExtra() *LoanExtra {
	return &LoanExtra{
		Loan: *l,
	}
}
func (l *LoanExtra) ToLoan() *Loan {
	return &l.Loan
}

func (l *LoanExtra) SetSchedules(sl []Schedule) {
	l.Schedules = sl
}
func (l *LoanExtra) AddSchedule(sl Schedule) {
	l.Schedules = append(l.Schedules, sl)
}

func (l *LoanExtra) SetRepayments(r []Repayment) {
	l.Repayments = r
}
func (l *LoanExtra) AddRepayment(repayment Repayment) {
	if l.Repayments == nil {
		l.Repayments = []Repayment{}
	}
	l.Repayments = append(l.Repayments, repayment)
}
func (l *LoanExtra) SetOverdueRecords(o []OverdueRecord) {
	l.OverdueRecords = o
}

func (l *LoanExtra) AddOverdueRecord(overdue OverdueRecord) {
	if l.OverdueRecords == nil {
		l.OverdueRecords = []OverdueRecord{}
	}
	l.OverdueRecords = append(l.OverdueRecords, overdue)
}

// OutstandingPrincipal 计算剩余本金（按 Schedules 未还本金累加）
func (l *LoanExtra) OutstandingPrincipal() decimal.Decimal {
	var sum decimal.Decimal
	for _, s := range l.Schedules {
		if s.Status == ScheduleUnpaid {
			sum = sum.Add(s.Principal)
		}
	}
	return sum
}
func (l *LoanExtra) OutstandingPeriods() int {
	var sum int
	for _, s := range l.Schedules {
		if s.Status == ScheduleUnpaid {
			sum++
		}
	}
	return sum
}

// NextUnpaidPeriod 返回下一个未还期序号（从 0 开始），若已全部还清返回 -1
func (l *LoanExtra) NextUnpaidPeriod() int {
	if l.Schedules == nil || len(l.Schedules) == 0 {
		return -1
	}
	for i, s := range l.Schedules {
		if s.Status != SchedulePaid && s.Status != ScheduleRemoved {
			return i
		}
	}
	return -1

}

// IsFullyPaid 是否结清
func (l *LoanExtra) IsFullyPaid() bool {
	return l.NextUnpaidPeriod() == 0
}

// PeriodRate 返回已经换算好的期别利率（领域服务可调用）
func (l *LoanExtra) PeriodRate() (decimal.Decimal, error) {
	return AnnualToPeriodRate(l.Product.Interest, l.Product.PeriodType, l.Product.DayCountConv)
}

func (l *LoanExtra) HasOverdue() bool {
	return l.OverdueRecords != nil && len(l.OverdueRecords) != 0
}

type OverdueRecord struct {
	ID             int64           `db:"id"`
	LoanID         int64           `db:"loan_id"`
	Period         int             `db:"period"`
	StartDate      time.Time       `db:"start_date"`
	DaysOver       int             `db:"days_over"`
	PenaltyAccrued decimal.Decimal `db:"penalty_accrued"` // 已计提罚息（原PenaltyAmt）
	PenaltyPaid    decimal.Decimal `db:"penalty_paid"`    // 已还罚息
	Rate           decimal.Decimal `db:"rate"`
	UpdatedAt      time.Time       `db:"updated_at"`
	Statue         OverdueStatus   `db:"statue"`
}

func NewOverdueRecord(id, loanId int64, period int, daysOver int, rate, penaltyAmt decimal.Decimal) *OverdueRecord {
	return &OverdueRecord{
		ID:             id,
		LoanID:         loanId,
		Period:         period,
		StartDate:      time.Now(),
		PenaltyAccrued: penaltyAmt,
		PenaltyPaid:    decimal.Zero,
		Rate:           rate,
		DaysOver:       0,
		UpdatedAt:      time.Now(),
		Statue:         OverdueStatusAccruing,
	}
}
func (o *OverdueRecord) TryToPay(amount decimal.Decimal) decimal.Decimal {
	if o.Statue == OverdueStatusCleared {
		return amount
	}
	p := o.PenaltyAccrued.Sub(o.PenaltyPaid)
	if amount.Cmp(p) == -1 {
		o.PenaltyPaid.Add(amount)
		amount = decimal.Zero
		o.Statue = OverdueStatusCleared
		o.UpdatedAt = time.Now()
		return amount
	} else {
		r := amount.Sub(p)
		amount = r
		o.Statue = OverdueStatusCleared
		o.UpdatedAt = time.Now()
		return amount
	}
}

type Product struct {
	ID             int64           `db:"id"  json:"id,omitempty"`
	Name           string          `db:"name" json:"name,omitempty"`
	Interest       decimal.Decimal `db:"interest" json:"interest"`
	MinPrinciple   decimal.Decimal `db:"min_principle" json:"min_principle"`
	MaxPrinciple   decimal.Decimal `db:"max_principle" json:"max_principle"`
	MinPeriods     int             `db:"min_periods" json:"min_periods,omitempty"`
	MaxPeriods     int             `db:"max_periods" json:"max_periods,omitempty"`
	RepayType      RepayType       `db:"repay_type" json:"repay_type,omitempty"`
	RollConvention RollConvention  `db:"roll_convention" json:"roll_convention,omitempty"`
	DayCountConv   DayCountConv    `db:"day_count_conv" json:"day_count_conv,omitempty"`
	PeriodType     PeriodType      `db:"period_type" json:"period_type,omitempty"`
	GraceTerm      int             `db:"grace_term" json:"grace_term,omitempty"` //宽限期，用于支持气球贷
	GraceDay       int             `db:"grace_day" json:"grace_day,omitempty"`   //允许的延迟还款日，在这几天内还款不算逾期
	Penalty        decimal.Decimal `db:"penalty" json:"penalty"`                 //逾期利率 TODO: 逾期利率支持阶梯
	DefaultRate    decimal.Decimal `db:"default_rate" json:"default_rate"`       //违约金，这玩意按道理也是该支持阶梯的
	Fees           []Fee           `db:"fees" json:"fees,omitempty"`
	Info           string          `db:"info" json:"info,omitempty"`
	extra          string          `db:"extra" json:"extra,omitempty"`
	Status        ProductStatues  `db:"status" json:"status,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

func (s *Product) GetName() string {
	return s.Name
}
func NewStrategy(
	name string,
	interest decimal.Decimal,
	minPrinciple decimal.Decimal,
	maxPrinciple decimal.Decimal,
	minPeriods int,
	maxPeriods int,
	repayType RepayType,
	rollConvention RollConvention,
	dayCountConv DayCountConv,
	periodType PeriodType,
	GraceTerm int,
	GraceDay int,
	penalty decimal.Decimal,
	fees []Fee,
	info string,
) *Product {
	return &Product{
		Name:           name,
		Interest:       interest,
		MinPrinciple:   minPrinciple,
		MaxPrinciple:   maxPrinciple,
		MinPeriods:     minPeriods,
		MaxPeriods:     maxPeriods,
		RepayType:      repayType,
		RollConvention: rollConvention,
		DayCountConv:   dayCountConv,
		PeriodType:     periodType,
		GraceTerm:      GraceTerm,
		GraceDay:       GraceDay,
		Penalty:        penalty,
		Fees:           fees,
		Info:           info,
	}
}
func (s *Product) isExist() bool {
	return true
}

type RepayInfo struct {
	Amount decimal.Decimal
	PrepayStrategy
}

// Repayment 一笔「用户实际还款」事件（这部分内容暂时这样处理，将来会加入具体的核销明细）
type Repayment struct {
	ID           int64           `db:"id"`
	LoanID       int64           `db:"loan_id"`
	RepayAt      time.Time       `db:"repay_at"`      // 到账时间（不是用户点击时间）
	TotalAmount  decimal.Decimal `db:"total_amount"`  // 用户实际支付总额（含所有费用）
	Status       RepayStatus     `db:"status"`        // SUCCESS / FAILED / CANCEL / REFUNDING /PROCESSING
	RefundAmount decimal.Decimal `db:"refund_amount"` // 已退金额（部分退、全额退）
	Extra        string          `db:"extra"`
}

func NewRepayment(id int64, loanID int64) *Repayment {
	return &Repayment{
		ID:           id,
		LoanID:       loanID,
		RepayAt:      time.Now(),
		Status:       RepayProcessing,
		TotalAmount:  decimal.Zero,
		RefundAmount: decimal.Zero,
	}
}
func (r *Repayment) AddAmount(amount decimal.Decimal) {
	r.TotalAmount = r.TotalAmount.Add(amount)
}

// Schedule 期供
type Schedule struct {
	ID               int64           `db:"id"`
	LoanID           int64           `db:"loan_id"`
	Period           int             `db:"period"`             // 第几期（从 1 开始）
	DueDate          time.Time       `db:"due_date"`           // 还款日
	Principal        decimal.Decimal `db:"principal"`          // 本期应还本金
	Interest         decimal.Decimal `db:"interest"`           // 本期应还利息
	ServiceFee       []Fee           `db:"service_fee"`        // 本期服务费（可扩展为多项费用）
	TotalPayment     decimal.Decimal `db:"total_payment"`      // 本期应还总额
	TotalPaymentPaid decimal.Decimal `db:"total_payment_paid"` //已支付的本期应还总额
	Status           ScheduleStatus  `db:"status"`             // 状态
	UpdatedAt        time.Time       `db:"updated_at"`
	Overdue          bool            `db:"overdue"`
}

// NewSchedule 工厂，保证金额 2 位小数
func NewSchedule(id, loanId int64, period int, dueDate time.Time, principal, interest decimal.Decimal, fee []Fee) *Schedule {
	total := principal.Add(interest)
	for _, s := range fee {
		s.Status = FeeStatusUnPaid
		total.Add(s.GetFee(principal))
	}

	return &Schedule{
		ID:               id,
		LoanID:           loanId,
		Period:           period,
		DueDate:          dueDate,
		Principal:        principal,
		Interest:         interest,
		ServiceFee:       fee,
		TotalPayment:     total,
		TotalPaymentPaid: decimal.Zero,
		Status:           ScheduleUnpaid,
		UpdatedAt:        time.Now(),
	}
}

// TryToPay 仅供正常还款使用，不考虑提前还款场景
func (s *Schedule) TryToPay(amount decimal.Decimal) decimal.Decimal {
	if s.Principal == decimal.Zero {
		s.Status = SchedulePaid
		return amount
	}
	if s.Status != SchedulePaid {
		f := s.TotalPayment.Sub(s.TotalPaymentPaid)
		if amount.Cmp(f) >= -1 {
			s.UpdatedAt = time.Now()
			s.Status = SchedulePaid
			for _, v := range s.ServiceFee {
				v.Status = FeeStatusPaid
			}
			return amount.Sub(f)
		} else {
			return s.tryToPay(amount)
		}
	} else {
		return amount
	}
}

func (s *Schedule) tryToPay(amount decimal.Decimal) decimal.Decimal {
	switch s.Status {
	case ScheduleUnpaid:
		f := s.Interest.Sub(s.TotalPaymentPaid)
		if amount.Cmp(f) >= 0 {
			s.TotalPaymentPaid.Add(f)
			amount = amount.Sub(f)
			s.Status = ScheduleInterestPaid
			return s.tryToPay(amount)
		} else {
			s.TotalPaymentPaid = s.TotalPaymentPaid.Add(amount)
			s.UpdatedAt = time.Now()
			return decimal.Zero
		}
	case ScheduleInterestPaid:
		if s.ServiceFee != nil && len(s.ServiceFee) > 0 {
			for _, v := range s.ServiceFee {
				if v.Status == FeeStatusPaid {
					continue
				} else {
					f := v.GetFee(s.Principal)
					if amount.Cmp(f) >= 0 {
						v.Status = FeeStatusPaid
						s.TotalPaymentPaid = s.TotalPaymentPaid.Add(f)
						amount = amount.Sub(f)
					} else {
						s.UpdatedAt = time.Now()
						return amount
					}
				}
			}
		}
		s.Status = ScheduleFeePaid
		return s.tryToPay(amount)
	case ScheduleFeePaid:
		f := s.TotalPayment.Sub(s.TotalPaymentPaid)
		if amount.Cmp(f) >= 0 {
			s.TotalPaymentPaid.Add(f)
			amount = amount.Sub(f)
			s.Status = SchedulePaid
			s.UpdatedAt = time.Now()
			return amount
		} else {
			s.TotalPaymentPaid = s.TotalPaymentPaid.Add(amount)
			s.UpdatedAt = time.Now()
			return decimal.Zero
		}
	default:
		return amount
	}

}

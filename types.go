package loancalc

import "github.com/shopspring/decimal"

// RepayType 还款方式
type RepayType string

// PeriodType 期别类型（用于把年化利率折算到对应期别）
type PeriodType string

// PrepayStrategy 提前还款策略
type PrepayStrategy string

// ------------------- 值对象 -------------------

type RoundStrategy = func(decimal decimal.Decimal) decimal.Decimal
type ScheduleStatus string
type OverdueStatus string
type FeeStatus string
type RepayStatus string

type DayCountConv string

type IDGenerator = func() int64

type RollConvention string
type ProductStatues string

type LoanStatus string

const (
	LoanPaid     LoanStatus = "PAID"
	LoanUnpaid   LoanStatus = "UNPAID"
	LoanPending  LoanStatus = "PENDING"
	LoanRejected LoanStatus = "REJECTED"
)

const (
	Unadjusted RollConvention = "UNADJUSTED"         //严格按日历算时间
	Following  RollConvention = "FOLLOWING"          //如果是节假日，向后挪
	Preceding  RollConvention = "PRECEDING"          //如果是节假日，向前挪
	ModFollow  RollConvention = "MODIFIED_FOLLOWING" //如果是节假日，向后挪，但如果跨月就向前挪
)

const (
	BONDBASIS   DayCountConv = "BONDBASIS"
	EUROBOND    DayCountConv = "EUROBOND"
	MONEYMARKET DayCountConv = "MONEYMARKET"
	FIXED       DayCountConv = "FIXED"
	ISDA        DayCountConv = "ISDA"
	AFB         DayCountConv = "AFB"
)

const (
	RepayTypeEqualPrincipal   RepayType = "EQUAL_PRINCIPAL"   // 等额本金
	RepayTypeEqualInstallment RepayType = "EQUAL_INSTALLMENT" // 等额本息
)

const (
	PeriodDay    PeriodType = "DAY"
	PeriodBiWeek PeriodType = "BI_WEEK"
	PeriodMonth  PeriodType = "MONTH"
	PeriodYear   PeriodType = "YEAR"
)

const (
	OverdueStatusAccruing OverdueStatus = "ACCRUING" // 仍在计息
	OverdueStatusPartial  OverdueStatus = "PARTIAL"  // 已部分还款
	OverdueStatusCleared  OverdueStatus = "CLEARED"  // 已全部结清
	OverdueStatusWaived   OverdueStatus = "WAIVED"   // 已减免
)
const (
	PrepayTermReduction    PrepayStrategy = "TERM_REDUCTION"    // 缩期
	PrepayPaymentReduction PrepayStrategy = "PAYMENT_REDUCTION" // 减供
	PrepayNot              PrepayStrategy = "NOT_PREPAY"        //按时还款
)

var BankRound = func(d decimal.Decimal) decimal.Decimal { return d.RoundBank(2) }

// Money 金额相加辅助，保持 2 位小数
func Money(d decimal.Decimal) decimal.Decimal {
	return d.RoundBank(2)
}

// TODO:将这里处理的更加语义化
const (
	ScheduleUnpaid       ScheduleStatus = "UNPAID"
	SchedulePaid         ScheduleStatus = "PAID"
	ScheduleInterestPaid ScheduleStatus = "INTEREST_PAID" //用来描述还款逾期但当期利息还清的状态
	ScheduleFeePaid      ScheduleStatus = "FEE_PAID"      //用来描述还款逾期，只有本金没有还清的状态
	ScheduleRemoved      ScheduleStatus = "REMOVED"
	SchedulePending      ScheduleStatus = "PENDING"
)

const (
	FeeStatusPaid    FeeStatus = "PAID"
	FeeStatusUnPaid  FeeStatus = "UNPAID"
	FeeStatuesModule FeeStatus = "MODULE" //模型类型，用于放置在产品中展示
)

const (
	RepaySuccess    RepayStatus = "SUCCESS"
	RepayFailed     RepayStatus = "FAILED"
	RepayCanceled   RepayStatus = "CANCELED"
	RepayProcessing RepayStatus = "PROCESSING"
	RepayRefunding  RepayStatus = "REFUNDING"
)

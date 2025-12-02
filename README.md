# LoanCalc - Go贷款计算库

一个功能完整的Go语言贷款计算库，支持多种还款方式、提前还款策略和复杂的计息规则。

## 特性

- 🏦 **多种还款方式**：支持等额本息、等额本金等常见还款方式
- 📅 **灵活的计息规则**：支持多种日期计算惯例和期别类型
- 💰 **提前还款策略**：支持缩期和减供两种提前还款策略
- 🔧 **插件化架构**：可扩展的插件系统，支持自定义业务逻辑
- 🎯 **精确计算**：使用decimal类型确保金融计算的精确性
- 📊 **完整的还款计划**：生成详细的还款计划表，支持费用和逾期管理

## 安装

```bash
go get github.com/Soulmate404/loancalc
```

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "time"
    "github.com/riskmanagement123/loancalc"
)

func main() {
    // 初始化引擎
    engine, err := loancalc.NewEngine(loancalc.Config{
        Clock: &loancalc.RealClock{},
    })
    if err != nil {
        panic(err)
    }

    // 创建产品
    product := &loancalc.Product{
        ID:             1,
        Name:           "个人消费贷款",
        Interest:       loancalc.DecimalFromFloat(0.05), // 年利率5%
        RepayType:      loancalc.RepayTypeEqualInstallment, // 等额本息
        PeriodType:     loancalc.PeriodMonth,
        DayCountConv:   loancalc.BONDBASIS,
        RollConvention: loancalc.Unadjusted,
        MinPeriods:     12,
        MaxPeriods:     360,
        GraceTerm:      0,
    }

    // 注册产品
    engine.RegisterProduct(product)

    // 创建贷款
    loan := loancalc.Loan{
        ID:           1,
        UserID:       1001,
        Principal:    loancalc.DecimalFromFloat(100000), // 10万元
        TotalPeriods: 36, // 3年
        Product:      product,
    }

    // 生成还款计划
    loanExtra, err := engine.BuildSchedules(loan)
    if err != nil {
        panic(err)
    }

    // 查看还款计划
    for _, schedule := range loanExtra.Schedules {
        fmt.Printf("期数: %d, 应还本金: %.2f, 应还利息: %.2f, 总额: %.2f, 到期日: %s\n",
            schedule.Period,
            schedule.Principal,
            schedule.Interest,
            schedule.TotalPayment,
            schedule.DueDate.Format("2006-01-02"),
        )
    }
}
```

### 还款操作

```go
// 正常还款
repayInfo := loancalc.RepayInfo{
    Amount:          loancalc.DecimalFromFloat(5000), // 还款金额
    PrepayStrategy:  loancalc.PrepayNot,              // 正常还款
    RepayAt:         time.Now(),
}

updatedLoan, remaining, err := engine.Repay(loanExtra, repayInfo)
if err != nil {
    panic(err)
}

fmt.Printf("剩余本金: %.2f\n", remaining)
```

### 提前还款

```go
// 提前还款 - 缩期
prepayInfo := loancalc.RepayInfo{
    Amount:          loancalc.DecimalFromFloat(20000), // 提前还款金额
    PrepayStrategy:  loancalc.PrepayTermReduction,    // 缩期
    RepayAt:         time.Now(),
}

updatedLoan, remaining, err := engine.Repay(loanExtra, prepayInfo)
if err != nil {
    panic(err)
}
```

## 核心概念

### 还款方式

- `EQUAL_INSTALLMENT`: 等额本息 - 每期还款金额相同
- `EQUAL_PRINCIPAL`: 等额本金 - 每期本金相同，利息递减

### 期别类型

- `DAY`: 日
- `BI_WEEK`: 双周
- `MONTH`: 月
- `YEAR`: 年

### 日期计算惯例

- `BONDBASIS`: 30/360
- `EUROBOND`: 30E/360
- `MONEYMARKET`: 实际天数/360
- `ISDA`: 实际天数/365
- `AFB`: 实际天数/365.25

### 滚动惯例

- `UNADJUSTED`: 不调整，严格按日历
- `FOLLOWING`: 遇节假日向后顺延
- `PRECEDING`: 遇节假日向前调整
- `MODIFIED_FOLLOWING`: 向后顺延但避免跨月

### 提前还款策略

- `TERM_REDUCTION`: 缩期 - 减少还款期数，月供不变
- `PAYMENT_REDUCTION`: 减供 - 期数不变，减少月供金额
- `NOT_PREPAY`: 正常还款

## 插件系统

LoanCalc支持插件扩展，可以在贷款创建和还款过程中注入自定义逻辑：

```go
type CustomPlugin struct{}

func (p *CustomPlugin) Name() string {
    return "CustomPlugin"
}

func (p *CustomPlugin) BeforeCreate(ctx *loancalc.LoanContext) error {
    // 贷款创建前的自定义逻辑
    return nil
}

func (p *CustomPlugin) AfterCreate(ctx *loancalc.LoanContext) error {
    // 贷款创建后的自定义逻辑
    return nil
}

// 注册插件
engine.RegisterProduct(product, &CustomPlugin{})
```

## 数据模型

### 产品 (Product)

定义贷款产品的核心参数，包括利率、期限、还款方式等。

### 贷款 (Loan/LoanExtra)

包含贷款的基本信息和完整的还款计划。

### 还款计划 (Schedule)

每期的详细还款信息，包括本金、利息、费用等。

### 还款记录 (Repayment)

记录每次还款的详细信息。

### 逾期记录 (OverdueRecord)

管理逾期状态和罚息计算。

## 注意事项

1. 所有金额计算使用`decimal.Decimal`类型，确保精度
2. 时间处理考虑了节假日和周末的调整
3. 支持复杂的费用结构和逾期处理
4. 插件系统需要在产品注册时一并注册

## 许可证

MIT Licence

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。


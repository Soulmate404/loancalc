package loancalc

import (
	"context"
	"errors"
)

type Plugin interface {
	Name() string
	BeforeCreate(ctx *LoanContext) error
	AfterCreate(ctx *LoanContext) error
}

type Plugins []Plugin

// LoanContext 内部上下文（对外不暴露内部池实现）
type LoanContext struct {
	Context context.Context
	Loan    *LoanExtra
	Params  map[string]any
}

// Engine 统一入口
type Engine struct {
	handlers map[int64]*handler
}

type handler struct {
	product   *Product
	plugins   Plugins
	buildFunc func(ctx *LoanContext) ([]Schedule, error)
	repayFunc func(ctx *LoanContext, info RepayInfo) (Decimal, error)
}

func NewEngine(c Config) (*Engine, error) {
	err := Start(c)
	return &Engine{handlers: make(map[int64]*handler)}, err
}

// RegisterProduct 绑定产品与插件链
func (e *Engine) RegisterProduct(p *Product, plugins ...Plugin) {
	h := &handler{product: p}
	if len(plugins) > 0 {
		h.plugins = append(h.plugins, plugins...)
	}
	// 默认核心处理流程：可根据产品类型定制
	h.buildFunc = func(ctx *LoanContext) ([]Schedule, error) {
		switch p.RepayType {
		case RepayTypeEqualInstallment:
			return AnnuitySchedule(ctx.Loan.ID, ctx.Loan.Principal, int64(ctx.Loan.TotalPeriods), p, cfg.IDGenerator)
		case RepayTypeEqualPrincipal:
			return EqualPrincipalSchedule(ctx.Loan.ID, ctx.Loan.Principal, int64(ctx.Loan.TotalPeriods), p, cfg.IDGenerator)
		default:
			return nil, ErrUnSupportRepayType
		}
	}
	h.repayFunc = func(ctx *LoanContext, info RepayInfo) (Decimal, error) {
		if info.PrepayStrategy == PrepayNot {
			return NormalRepay(ctx.Loan, info.Amount, cfg.IDGenerator)
		}
		return PreRepay(ctx.Loan, info.Amount, cfg.IDGenerator, info.PrepayStrategy)
	}
	e.handlers[p.ID] = h
}

// BuildSchedules 根据产品还款方式生成计划
func (e *Engine) BuildSchedules(l Loan) (*LoanExtra, error) {
	h, ok := e.handlers[l.Product.ID]
	if !ok {
		return nil, errors.New("product not registered")
	}
	ctx := &LoanContext{Context: context.Background(), Loan: l.ToLoanExtra(), Params: map[string]any{}}
	for _, p := range h.plugins {
		if err := p.BeforeCreate(ctx); err != nil {
			return nil, err
		}
	}
	schedules, err := h.buildFunc(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Loan.SetSchedules(schedules)
	for i := len(h.plugins) - 1; i >= 0; i-- {
		if err := h.plugins[i].AfterCreate(ctx); err != nil {
			return nil, err
		}
	}
	return ctx.Loan, nil
}

// Repay 统一还款入口
func (e *Engine) Repay(l *LoanExtra, info RepayInfo) (*LoanExtra, Decimal, error) {
	h, ok := e.handlers[l.Product.ID]
	if !ok {
		return nil, Decimal{}, errors.New("product not registered")
	}
	ctx := &LoanContext{Context: context.Background(), Loan: l, Params: map[string]any{}}
	for _, p := range h.plugins {
		if err := p.BeforeCreate(ctx); err != nil {
			return nil, Decimal{}, err
		}
	}
	remaining, err := h.repayFunc(ctx, info)
	if err != nil {
		return nil, Decimal{}, err
	}
	for i := len(h.plugins) - 1; i >= 0; i-- {
		if err := h.plugins[i].AfterCreate(ctx); err != nil {
			return nil, Decimal{}, err
		}
	}
	return ctx.Loan, remaining, nil
}

// SetHandlerFuncs 允许为指定产品自定义核心流程
func (e *Engine) SetHandlerFuncs(productID int64,
	build func(ctx *LoanContext) ([]Schedule, error),
	repay func(ctx *LoanContext, info RepayInfo) (Decimal, error),
) error {
	h, ok := e.handlers[productID]
	if !ok {
		return errors.New("product not registered")
	}
	if build != nil {
		h.buildFunc = build
	}
	if repay != nil {
		h.repayFunc = repay
	}
	return nil
}

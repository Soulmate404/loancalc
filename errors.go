package loancalc

import (
	"errors"
)

var (
	ErrNoScheduleFound         = errors.New("no schedule found")
	ErrTodayNotDueDate         = errors.New("today is not the due date")
	ErrInsufficientForPenalty  = errors.New("insufficient amount to cover penalty interest")
	ErrInsufficientForSchedule = errors.New("insufficient amount to cover schedule")
	ErrUnSupportRepayType      = errors.New("unsupported repay type")
)

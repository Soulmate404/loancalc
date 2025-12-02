package loancalc

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Clock 提供可替换的时间源
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// HolidayProvider 提供节假日判断
type HolidayProvider interface {
	IsHoliday(t time.Time) bool
}

// ChinaHolidayProvider 默认中国节假日+周末
type ChinaHolidayProvider struct{}

func (ChinaHolidayProvider) IsHoliday(t time.Time) bool {
	// 复用内部实现，并在 Start 时预热 convention.Holiday
	d, ok := Holiday[t.Format("2006-01-02")]
	if ok && d {
		return true
	}
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// Config 运行时配置
type Config struct {
	IDGenerator   IDGenerator
	RoundStrategy RoundStrategy
	//TODO 舍入策略适配
	Holiday HolidayProvider
	Clock   Clock
}

var Holiday = map[string]bool{}
var cfg Config

// Start 初始化运行时配置与默认依赖。
func Start(c Config) error {
	if c.Clock == nil {
		c.Clock = systemClock{}
	}
	if c.RoundStrategy == nil {
		c.RoundStrategy = BankRound
	}
	if c.Holiday == nil {
		c.Holiday = ChinaHolidayProvider{}
	}
	if len(Holiday) == 0 {
		// 尝试拉取当年中国法定节假日，失败不视为致命错误
		if h, err := FetchCN(); err == nil {
			Holiday = h
		}
	}
	cfg = c
	return nil
}

// FetchChinaHolidays 允许调用端主动预热节假日数据
func FetchChinaHolidays() (map[string]bool, error) {
	h, err := FetchCN()
	if err != nil {
		return nil, err
	}
	Holiday = h
	return h, nil
}

// FetchCN 自动获取当年节假日+调休，返回 map[yyyy-mm-dd]bool，true 表示放假
func FetchCN() (map[string]bool, error) {
	type timorResp struct {
		Code    int
		Holiday map[string]struct {
			Date    string
			Holiday bool // true 放假 false 调休上班
			Name    string
			Wage    int    // 1 三倍工资
			After   bool   // true 节后补班
			Target  string // 对应节日
			Rest    string
		}
	}
	url := "https://timor.tech/api/holiday/year/"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	body, _ := io.ReadAll(resp.Body)

	var tr timorResp
	if err = json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	m := make(map[string]bool)
	for _, v := range tr.Holiday {
		m[v.Date] = v.Holiday // 只记录放假日期
	}
	return m, nil
}

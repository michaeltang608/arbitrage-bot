package oke

import "fmt"

func (s *Service) GetOrderStat() string {
	one, two, three, four := "nil", "nil", "nil", "nil"
	if s.openMarginOrder != nil {
		one = s.openMarginOrder.State
	}
	if s.openFutureOrder != nil {
		two = s.openFutureOrder.State
	}
	if s.closeMarginOrder != nil {
		three = s.closeMarginOrder.State
	}
	if s.closeFutureOrder != nil {
		four = s.closeFutureOrder.State

	}
	return fmt.Sprintf("%s-%s-%s-%s", one, two, three, four)
}

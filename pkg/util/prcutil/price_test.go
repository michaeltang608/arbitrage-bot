package prcutil

import (
	"log"
	"testing"
)

func TestA(t *testing.T) {
	adjustPrice := AdjustPrice(0.6793, "buy", 1.072545806643821)
	log.Printf("adjustPrice: %v\n", adjustPrice)
}

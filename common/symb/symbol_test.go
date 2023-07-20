package symb

import (
	"log"
	"strings"
	"testing"
)

func TestA(t *testing.T) {
	unionResult := make([]string, 0)
	for _, i := range marginList {
		for _, j := range futureList {
			if i == j {
				unionResult = append(unionResult, i)
			}
		}
	}
	log.Printf("unionResult: \n%v\n", strings.Join(unionResult, `", "`))
}

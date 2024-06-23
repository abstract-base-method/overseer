package common

import (
	"fmt"
	"testing"
)

func TestRangedInt(t *testing.T) {
	tests := []struct {
		minimum   float32
		selection float32
		maximum   float32
		maxInt    int
	}{
		{0.01, 0.01, 2.0, 256},
		{0.01, 0.2, 2.0, 256},
		{0.01, 0.8, 2.0, 256},
		{0.01, 1.0, 2.0, 256},
		{0.01, 1.5, 2.0, 256},
		{0.01, 1.8, 2.0, 256},
		{0.01, 2.0, 2.0, 256},
	}

	for _, tt := range tests {
		result := RandomizedProgressiveValue(tt.minimum, tt.selection, tt.maximum, tt.maxInt)
		fmt.Printf("inputs: min=%f selection=%f max=%f maxInt=%d -- result=%d\n", tt.minimum, tt.selection, tt.maximum, tt.maxInt, result)
		if result <= 0 || result > tt.maxInt {
			t.Errorf("rangedInt(%v, %v, %v, %v) = %v; want a value between 1 and %v", tt.minimum, tt.selection, tt.maximum, tt.maxInt, result, tt.maxInt)
		}
	}
}

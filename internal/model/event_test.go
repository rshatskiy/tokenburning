package model

import "testing"

func TestTokensTotalFallsBackToSum(t *testing.T) {
	tk := Tokens{Input: 10, Output: 5, CacheRead: 2}
	if got := tk.TotalOrSum(); got != 17 {
		t.Fatalf("TotalOrSum() = %d, want 17", got)
	}
	tk2 := Tokens{Total: 100, Input: 10}
	if got := tk2.TotalOrSum(); got != 100 {
		t.Fatalf("TotalOrSum() with explicit Total = %d, want 100", got)
	}
}

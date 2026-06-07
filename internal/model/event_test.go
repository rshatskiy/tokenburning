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
	if got := (Tokens{}).TotalOrSum(); got != 0 {
		t.Fatalf("zero-value TotalOrSum() = %d, want 0", got)
	}
	tk3 := Tokens{Input: 1, Output: 2, CacheRead: 3, Cache1h: 4, Cache5m: 5, Reasoning: 6}
	if got := tk3.TotalOrSum(); got != 21 {
		t.Fatalf("all-components TotalOrSum() = %d, want 21", got)
	}
}

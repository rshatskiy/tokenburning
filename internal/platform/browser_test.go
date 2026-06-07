package platform

import (
	"runtime"
	"testing"
)

func TestBrowserCommandForOS(t *testing.T) {
	name, args := browserCommand("http://127.0.0.1:1234/?t=x")
	if name == "" {
		t.Fatal("пустая команда открытия браузера")
	}
	if len(args) == 0 || args[len(args)-1] != "http://127.0.0.1:1234/?t=x" {
		t.Fatalf("URL не передан в аргументы: %v %v", name, args)
	}
	// sanity по текущей ОС
	switch runtime.GOOS {
	case "darwin":
		if name != "open" {
			t.Fatalf("darwin: name=%q, want open", name)
		}
	case "linux":
		if name != "xdg-open" {
			t.Fatalf("linux: name=%q, want xdg-open", name)
		}
	}
}

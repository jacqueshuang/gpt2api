package main

import "testing"

func TestOpenAIAddrDefaultHost(t *testing.T) {
	got := openAIAddr("", 17200)
	if got != ":17200" {
		t.Fatalf("openAIAddr empty host = %q, want %q", got, ":17200")
	}
}

func TestOpenAIAddrLocalhost(t *testing.T) {
	got := openAIAddr("127.0.0.1", 17201)
	if got != "127.0.0.1:17201" {
		t.Fatalf("openAIAddr localhost = %q, want %q", got, "127.0.0.1:17201")
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestWidgetFrameAncestors(t *testing.T) {
	want := []string{
		"https://sisol.olcepks.ca",
		"http://sisol.olcepks.ca",
		"https://localhost:6200",
		"http://localhost:6200",
		"https://*.example.com",
		"http://*.example.com",
	}
	got := widgetFrameAncestors([]string{"sisol.olcepks.ca", " localhost:6200 ", "", "*.example.com"})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("widgetFrameAncestors() = %#v, want %#v", got, want)
	}
}

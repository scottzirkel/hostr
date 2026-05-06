package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestStreamDevOutputDetectsCommonLocalURLs(t *testing.T) {
	out, err := os.CreateTemp(t.TempDir(), "dev-output-*")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	ports := make(chan int, 8)
	input := strings.Join([]string{
		"Local:   http://localhost:5173/",
		"Network: http://127.0.0.1:3000/dashboard",
		"Listening on http://0.0.0.0:8080",
		"IPv6:    http://[::1]:4000/",
		"ignored: http://localhost:65536/",
	}, "\n")

	streamDevOutput(strings.NewReader(input), out, ports)
	close(ports)

	var got []int
	for port := range ports {
		got = append(got, port)
	}
	want := []int{5173, 3000, 8080, 4000}
	if len(got) != len(want) {
		t.Fatalf("ports = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ports = %#v, want %#v", got, want)
		}
	}
}

package client

import (
	"testing"
)

func TestEcho(t *testing.T) {
	msg := "message"
	ret, err := r.Echo(msg)
	if err != nil {
		t.Error(err)
	} else if ret != msg {
		t.Errorf("echo %s\n%s", msg, ret)
	}
}

func TestPing(t *testing.T) {
	if err := r.Ping(); err != nil {
		t.Error(err)
	}
}

func BenchmarkPing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		r.Ping()
	}
}

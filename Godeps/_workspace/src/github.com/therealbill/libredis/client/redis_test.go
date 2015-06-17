package client

import (
	"fmt"
	"testing"
	"time"
)

var (
	network  = "tcp"
	address  = "127.0.0.1:6379"
	db       = 1
	password = ""
	timeout  = 5 * time.Second
	maxidle  = 1
	r        *Redis

	format = "tcp://auth:%s@%s/%d?timeout=%s&maxidle=%d"
)

func init() {
	client, err := DialWithConfig(&DialConfig{network, address, db, password, timeout, maxidle})
	if err != nil {
		panic(err)
	}
	r = client
}

func TestDial(t *testing.T) {
	redis, err := DialWithConfig(&DialConfig{network, address, db, password, timeout, maxidle})
	if err != nil {
		t.Error(err)
	} else if err := redis.Ping(); err != nil {
		t.Error(err)
	}
	redis.pool.Close()
}

func TestDialTimeout(t *testing.T) {
	redis, err := DialWithConfig(&DialConfig{network, address, db, password, timeout, maxidle})
	if err != nil {
		t.Error(err)
	} else if err := redis.Ping(); err != nil {
		t.Error(err)
	}
	redis.pool.Close()
}

func TestDiaURL(t *testing.T) {
	redis, err := DialURL(fmt.Sprintf(format, password, address, db, timeout.String(), maxidle))
	if err != nil {
		t.Fatal(err)
	} else if err := redis.Ping(); err != nil {
		t.Error(err)
	}
	redis.pool.Close()
}

func TestPackCommand(t *testing.T) {
	res_string := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	res, err := packCommand("SET", "key", "value")
	if err != nil {
		t.Error(err)
	}
	if string(res) != res_string {
		fmt.Printf("res='%s', should be '%s'", string(res), res_string)
		t.Error("packCommand not working")
	}
}

func BenchmarkPackCommandString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := packCommand("SET", "key", "value")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPackCommandInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := packCommand("SET", "key", i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPackCommandInt64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := packCommand("SET", "key", int64(i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPackCommandFloat64(b *testing.B) {
	var val float64 = 2.0
	for i := 0; i < b.N; i++ {
		_, err := packCommand("SET", "key", val)
		if err != nil {
			b.Fatal(err)
		}
	}
}

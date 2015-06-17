package airbrake

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"
)

const API_KEY = ""

func TestError(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"

	err := Error(errors.New("Test Error"), nil)
	if err != nil {

		t.Error(err)
	}

	time.Sleep(1e9)
}

func TestRequest(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"

	request, _ := http.NewRequest("GET", "/some/path?a=1", bytes.NewBufferString(""))

	err := Error(errors.New("Test Error"), request)

	if err != nil {
		t.Error(err)
	}

	time.Sleep(1e9)
}

func TestNotify(t *testing.T) {
	Verbose = true
	ApiKey = API_KEY
	Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"
	
	err := Notify(errors.New("Test Error"))
	
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1e9)
}

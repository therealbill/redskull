package airbrake

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"sync"
	"text/template"
)

var (
	ApiKey      = ""
	Endpoint    = "https://api.airbrake.io/notifier_api/v2/notices"
	Environment = "development"
	Verbose     = false

	badResponse   = errors.New("Bad response")
	apiKeyMissing = errors.New("Please set the airbrake.ApiKey before doing calls")
	dunno         = []byte("???")
	centerDot     = []byte("·")
	dot           = []byte(".")
	tmpl          = template.Must(template.New("error").Parse(source))
)

type Line struct {
	Function string
	File     string
	Line     int
}

// stack implements Stack, skipping N frames
func stacktrace(skip int) (lines []Line) {
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		item := Line{string(function(pc)), string(file), line}

		// ignore panic method
		if item.Function != "panic" {
			lines = append(lines, item)
		}
	}
	return
}

var channel chan map[string]interface{}
var once sync.Once

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//  runtime/debug.*T·ptrmethod
	// and want
	//  *T.ptrmethod
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

func initChannel() {
	channel = make(chan map[string]interface{}, 100)

	go func() {
		for params := range channel {
			post(params)
		}
	}()
}

func post(params map[string]interface{}) {
	buffer := bytes.NewBufferString("")

	if err := tmpl.Execute(buffer, params); err != nil {
		log.Printf("Airbrake error: %s", err)
		return
	}

	if Verbose {
		log.Printf("Airbrake payload for endpoint %s: %s", Endpoint, buffer)
	}

	response, err := http.Post(Endpoint, "text/xml", buffer)
	if err != nil {
		log.Printf("Airbrake error: %s", err)
		return
	}

	if Verbose {
		body, _ := ioutil.ReadAll(response.Body)
		log.Printf("response: %s", body)
	}
	response.Body.Close()

	if Verbose {
		log.Printf("Airbrake post: %s status code: %d", params["Error"], response.StatusCode)
	}

}

func Error(e error, request *http.Request) error {
	once.Do(initChannel)

	if ApiKey == "" {
		return apiKeyMissing
	}

	params := map[string]interface{}{
		"Class":       reflect.TypeOf(e).String(),
		"Error":       e,
		"ApiKey":      ApiKey,
		"ErrorName":   e.Error(),
		"Environment": Environment,
		"Request":     request,
	}

	if params["Class"] == "" {
		params["Class"] = "Panic"
	}

	pwd, err := os.Getwd()
	if err == nil {
		params["Pwd"] = pwd
	}

	params["Backtrace"] = stacktrace(3)

	post(params)

	return nil
}

type ExtendedNotification struct {
	ErrorClass string
	ErrorName  string
	Error      error
}

func ExtendedError(notice ExtendedNotification, req *http.Request) error {
	once.Do(initChannel)
	if ApiKey == "" {
		return apiKeyMissing
	}
	log.Printf("%+v", notice)
	params := map[string]interface{}{
		"Class":       notice.ErrorClass,
		"Error":       notice.Error,
		"ApiKey":      ApiKey,
		"ErrorName":   notice.Error.Error(),
		"Environment": Environment,
	}
	if params["Class"] == "" {
		params["Class"] = "Panic"
	}

	pwd, err := os.Getwd()
	if err == nil {
		params["Pwd"] = pwd
	}

	hostname, err := os.Hostname()
	if err == nil {
		params["Hostname"] = hostname
	}
	params["Backtrace"] = stacktrace(3)
	post(params)
	return nil
}

func Notify(e error) error {
	once.Do(initChannel)

	if ApiKey == "" {
		return apiKeyMissing
	}

	params := map[string]interface{}{
		"Class":       reflect.TypeOf(e).String(),
		"Error":       e,
		"ApiKey":      ApiKey,
		"ErrorName":   e.Error(),
		"Environment": Environment,
	}

	if params["Class"] == "" {
		params["Class"] = "Panic"
	}

	pwd, err := os.Getwd()

	if err == nil {
		params["Pwd"] = pwd
	}

	hostname, err := os.Hostname()

	if err == nil {
		params["Hostname"] = hostname
	}

	params["Backtrace"] = stacktrace(3)

	post(params)
	return nil

}

func CapturePanic(r *http.Request) {
	if rec := recover(); rec != nil {

		if err, ok := rec.(error); ok {
			log.Printf("Recording err %s", err)
			Error(err, r)
		} else if err, ok := rec.(string); ok {
			log.Printf("Recording string %s", err)
			Error(errors.New(err), r)
		}

		panic(rec)
	}
}

const source = `<?xml version="1.0" encoding="UTF-8"?>
<notice version="2.0">
  <api-key>{{ .ApiKey }}</api-key>
  <notifier>
    <name>Airbrake Golang</name>
    <version>0.0.1</version>
    <url>http://airbrake.io</url>
  </notifier>
  <error>
    <class>{{ html .Class }}</class>
    <message>{{ with .ErrorName }}{{html .}}{{ end }}</message>
    <backtrace>
      {{ range .Backtrace }}
      <line method="{{ html .Function}}" file="{{ html .File}}" number="{{.Line}}"/>
      {{ end }}
    </backtrace>
  </error>
  {{ with .Request }}
  <request>
    <url>{{ html .URL }}</url>
    <component/>
    <action/>
  </request>
  {{ end }}  
  <server-environment>
    <project-root>{{ html .Pwd }}</project-root>   
    <environment-name>{{ .Environment }}</environment-name>
    <hostname>{{ html .Hostname }}</hostname>
  </server-environment>
</notice>`

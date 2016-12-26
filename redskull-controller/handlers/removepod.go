package handlers

import (
	"log"
	"net/http"

	"github.com/zenazn/goji/web"
)

// RemovePodHTML is the action target for adding a pod. It does the heavy lifting
func RemovePodHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	log.Print("########### REMOVE POD PROCESSING ###########")
	context, err := NewPageContext()
	checkContextError(err, &w)
	context.Title = "Pod Remove Result"
	context.ViewTemplate = "removepod"
	type results struct {
		Name     string
		Message  string
		Error    string
		HasError bool
	}
	podname := c.URLParams["podname"]
	res := results{Name: podname}

	_, err = context.Constellation.RemovePod(podname)
	if err != nil {
		log.Printf("Error on remove pod: %s", err.Error())
		res.Message = "Error on attempt to remove pod"
		res.Error = err.Error()
		res.HasError = true
		return
	} else {
		res.Message = "Pod " + podname + " was removed from management"
	}
	context.Data = res
	log.Print("########### REMOVE POD PROCESSED ###########")
	render(w, context)
}

package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
)

// ErrorMetrics is a struct used by the UI to display the current breakdown of
// errors among the pod.
type ErrorMetrics struct {
	NoQuorum         int
	MissingSentinels int
	TooManySentinels int
	NoValidSlave     int
	InvalidAuth      int
	TotalErrorPods   int
	ConnectionError  int
	NoFailover       int
	Groups           map[string][]interface{}
}

// Dashboard shows the dashboard
func Dashboard(c web.C, w http.ResponseWriter, r *http.Request) {
	dash_start := time.Now()
	log.Print("Dashboard requested %v", dash_start)
	context, err := NewPageContext()
	checkContextError(err, &w)
	context.ViewTemplate = "dashboard"
	context.Title = "RedSkull: Dashboard"
	context.Refresh = true
	context.RefreshURL = r.URL.Path
	context.RefreshTime = 60
	log.Printf("Dashboard context set up %v from dash call", time.Since(dash_start))

	var emet ErrorMetrics
	errgroups := make(map[string][]interface{})
	log.Print("dashboard calling GetPodsInError")
	pods := context.Constellation.GetPodsInError()
	log.Printf("Dashboard error pod call %v from dash call", time.Since(dash_start))
	emet.TotalErrorPods = len(pods)
	counted := make(map[string]interface{})
	for _, pod := range pods {
		if pod.Name == "" {
			continue
		}
		_, dupe := counted[pod.Name]
		if dupe {
			log.Print("Mice!? IN MY BAGUETTES?!")
			continue
		}
		if pod.MissingSentinels {
			emet.MissingSentinels++
		}
		if pod.TooManySentinels {
			emet.TooManySentinels++
		}
		if !pod.HasQuorum() {
			emet.NoQuorum++
		}
		if !pod.CanFailover() {
			emet.NoFailover++
		}
		if pod.Master == nil {
			emet.ConnectionError++
		} else {
			if !pod.Master.HasValidAuth {
				emet.InvalidAuth++
				pod.ValidAuth = false
				errgroups["InvalidAuth"] = append(errgroups["InvalidAuth"], pod)
			} else if pod.Master.Info.Replication.ConnectedSlaves == 0 || len(pod.Master.Slaves) == 0 || !pod.HasValidSlaves {
				emet.NoValidSlave++
			}
		}
	}
	emet.Groups = errgroups
	log.Printf("NoAuth: %d", len(errgroups["InvalidAuth"]))
	log.Printf("Dashboard iterated over pods in error in  %v from dash call", time.Since(dash_start))
	context.Data = emet
	render(w, context)
	log.Printf("Dashboard completed  %v from dash call", time.Since(dash_start))
}

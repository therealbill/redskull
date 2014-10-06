package handlers

import (
	"net/http"

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
}

// Dashboard shows the dashboard
func Dashboard(c web.C, w http.ResponseWriter, r *http.Request) {
	context := NewPageContext()
	context.ViewTemplate = "dashboard"
	context.Title = "RedSkull: Dashboard"
	context.Refresh = true
	context.RefreshURL = r.URL.Path
	context.RefreshTime = 60

	var emet ErrorMetrics
	pods := context.Constellation.GetPodsInError()
	emet.TotalErrorPods = len(pods)
	for _, pod := range pods {
		if pod.Name == "" {
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
		if pod.Master == nil {
			emet.NoValidSlave++
		} else {
			if pod.Master.Info.Replication.ConnectedSlaves == 0 || len(pod.Master.Slaves) == 0 || !pod.HasValidSlaves {
				emet.NoValidSlave++
			}
		}
		if !pod.HasInfo && !pod.ValidAuth {
			emet.InvalidAuth++
		}
	}
	context.Data = emet
	render(w, context)
}

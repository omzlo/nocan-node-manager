package controller

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"strings"
)

type Application struct {
	Router     *httprouter.Router
	Channels   *ChannelController
	Nodes      *NodeController
	Interfaces *InterfaceController
	Jobs       *JobController
}

func NewApplication() *Application {
	app := &Application{}
	app.Router = httprouter.New()
	app.Channels = NewChannelController()
	app.Nodes = NewNodeController()
	app.Interfaces = NewInterfaceController()
	app.Jobs = NewJobController()
	return app
}

func (app *Application) Run() {
	go http.ListenAndServe(":8888", &CheckRouter{app.Router})
	go model.Channels.Run()
	go model.Interfaces.Run()
	go model.Jobs.Run()
	model.Nodes.Run()
}

/****/

type CheckRouter struct {
	handler http.Handler
}

func (cr *CheckRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clog.Debug("%s request from %s to %s", r.Method, r.RemoteAddr, r.RequestURI)
	if r.URL.Path != "/" /*&& r.URL.Path != "/static/"*/ {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
	}
	if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		r.ParseForm()
		if r.Form.Get("_method") == http.MethodPut {
			r.Method = http.MethodPut
		}
	}
	cr.handler.ServeHTTP(w, r)
}

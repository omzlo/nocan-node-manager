package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	//	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"strings"
)

type MainTask struct {
	BaseTask
	//PortManager *model.PortManager
	//Port        *model.Port
	Router  *httprouter.Router
	Topics  *TopicController
	Nodes   *NodeController
	Drivers *DriverController
}

func NewMainTask(pm *model.PortManager) *MainTask {
	app := &MainTask{}
	app.PortManager = pm
	app.Router = httprouter.New()
	app.Nodes = NewNodeController(app.PortManager, "nodes.dat")
	app.Topics = NewTopicController(app.PortManager, app.Nodes.Model)
	app.Drivers = NewDriverController(app.PortManager)
	return app
}

func (app *MainTask) ProcessRecv(port *model.Port) {
	<-port.Input
}

func (app *MainTask) Run() {
	go http.ListenAndServe(":8888", &CheckRouter{app.Router})
	go app.Topics.Run()
	go app.Drivers.Run()
	app.Nodes.Run()
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

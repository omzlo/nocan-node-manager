package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/clog"
	"strings"
)

type ApplicationController struct {
	Port
	Router *httprouter.Router
	Ports  *PortController
	Topics *TopicController
	Nodes  *NodeController
}

func NewApplication() *ApplicationController {
	app := &ApplicationController{}
	app.Router = httprouter.New()
	app.Topics = NewTopicController(app)
	app.Nodes = NewNodeController(app)
	app.Ports = NewPortController(app)
	app.Port = app.Ports.Model.Add(NewCoreEndpoint(app))
	app.Ports.Model.Add(NewLogEndpoint(app))
	return app
}

func (app *ApplicationController) Run() error {
	go app.Ports.Model.ListenAndServe()
	return http.ListenAndServe(":8888", &CheckRouter{app.Router})
}

func (app *ApplicationController) SendMessage(m *Message) {
	app.Ports.Model.SendMessage(app.Port, m)
}

func (app *ApplicationController) Publish(node Node, topic Topic, data []byte) {
	//clog.Debug("Publish node=%d, topic=%d dlen=%d", int(node), int(topic), len(data))
	m := NewPublishMessage(node, topic, data)
	clog.Debug("Publish %s", m.String())
	app.SendMessage(m)
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
	cr.handler.ServeHTTP(w, r)
}

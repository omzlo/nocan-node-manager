package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"strings"
)

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I hate %s!", r.URL.Path[1:])
}

type CheckRouter struct {
	handler http.Handler
}

func (cr *CheckRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clog.Debug("%s request from %s to %s", r.Method, r.RemoteAddr, r.RequestURI)
	if r.URL.Path != "/" && r.URL.Path != "/static/" {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
	}
	cr.handler.ServeHTTP(w, r)
}

type LogFileSystem struct{}

func (lf *LogFileSystem) Open(name string) (http.File, error) {
	clog.Debug("HTTP LOG: %s", name)
	return nil, fmt.Errorf("Logged as %s", name)
}

func main() {
	var id [8]byte

	fmt.Println("Start")

	//if se != nil {
	//	se.Close()
	//}

	data, _ := ioutil.ReadFile("test.hex")

	ih := intelhex.New()
	err := ih.Load(strings.NewReader(string(data)))
	if err != nil {
		clog.Error("%s", err.Error())
	}

	app := nocan.NewApplication()
	app.Topics.Model.Register("/clock")
	app.Topics.Model.Register("pizza")
	nocan.StringToUid("01:02:03:04:05:06:07:88", id[:])
	app.Nodes.Model.Register(id[:])

	se := nocan.NewSerialEndpoint("/dev/cu.usbmodem12341")
	if se != nil {
		app.Ports.Model.Add(se)
	}

	homepage := nocan.NewHomePageController()
	nodepage := nocan.NewNodePageController()

	app.Router.GET("/api/topics", app.Topics.Index)
	app.Router.GET("/api/topics/*topic", app.Topics.Show)
	app.Router.PUT("/api/topics/*topic", app.Topics.Update)
	app.Router.GET("/api/nodes", app.Nodes.Index)
	app.Router.GET("/api/nodes/:node", app.Nodes.Show)
	app.Router.GET("/api/ports", app.Ports.Index)
	app.Router.ServeFiles("/static/*filepath", http.Dir("../static"))
	app.Router.GET("/nodes", nodepage.Index)
	app.Router.GET("/", homepage.Index)

	app.Run()
	fmt.Println("Done")
}

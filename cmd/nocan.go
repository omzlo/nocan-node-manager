package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"pannetrat.com/nocan/model"
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

	main := nocan.NewMainTask()
	main.Topics.Model.Register("/clock")
	main.Topics.Model.Register("pizza")
	model.StringToUid("01:02:03:04:05:06:07:88", id[:])
	main.Nodes.Model.Register(id[:])

	se := nocan.NewSerialTask("/dev/cu.usbmodem12341")
	if se != nil {
		main.TaskManager.CreateTask("serial", se)
	}
	main.TaskManager.CreateTask("log", nocan.NewLogTask())

	homepage := nocan.NewHomePageController()
	nodepage := nocan.NewNodePageController()

	main.Router.GET("/api/topics", main.Topics.Index)
	main.Router.GET("/api/topics/*topic", main.Topics.Show)
	main.Router.PUT("/api/topics/*topic", main.Topics.Update)
	main.Router.GET("/api/nodes", main.Nodes.Index)
	main.Router.GET("/api/nodes/:node", main.Nodes.Show)
	main.Router.GET("/api/nodes/:node/flash", main.Nodes.Flash.Show)
	//main.Router.GET("/api/ports", main.Ports.Index)
	main.Router.ServeFiles("/static/*filepath", http.Dir("../static"))
	main.Router.GET("/nodes", nodepage.Index)
	main.Router.GET("/", homepage.Index)

	main.TaskManager.Run()
	fmt.Println("Done")
}

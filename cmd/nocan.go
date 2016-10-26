package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
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

	core := nocan.NewCoreEndpoint()
	core.Topics.Model.Register("/clock")
	core.Topics.Model.Register("pizza")
	nocan.StringToUid("01:02:03:04:05:06:07:88", id[:])
	core.Nodes.Model.Register(id[:])

	core.Ports.Model.Add(&nocan.LogEndpoint{})
	se := nocan.NewSerialEndpoint("/dev/cu.usbmodem12341")
	if se != nil {
		core.Ports.Model.Add(se)
	}

	homepage := nocan.NewHomePageController()
	nodepage := nocan.NewNodePageController()

	router := httprouter.New()
	router.GET("/api/topics", core.Topics.Index)
	router.GET("/api/topics/*topic", core.Topics.Show)
	router.PUT("/api/topics/*topic", core.Topics.Update)
	router.GET("/api/nodes", core.Nodes.Index)
	router.GET("/api/nodes/:node", core.Nodes.Show)
	router.GET("/api/ports", core.Ports.Index)
	router.ServeFiles("/static/*filepath", http.Dir("../static"))
	router.GET("/nodes", nodepage.Index)
	router.GET("/", homepage.Index)

	core.ListenAndServe()
	http.ListenAndServe(":8888", &CheckRouter{router})
	fmt.Println("Done")
}

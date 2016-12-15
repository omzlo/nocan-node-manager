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

	portmanager := model.NewPortManager()

	main := nocan.NewMainTask(portmanager)
	main.Topics.Model.Register("/clock")
	main.Topics.Model.Register("pizza")
	model.StringToUdid("01:02:03:04:05:06:07:88", id[:])
	main.Nodes.Model.Register(id[:])

	driver, err := model.NewDriver("/dev/cu.usbmodem12341")
	if err != nil {
		panic(err)
	}
	main.Drivers.Model.Add(driver)

	lt := nocan.NewLogTask(portmanager)
	if lt != nil {
		go lt.Run()
	}

	homepage := nocan.NewHomePageController()
	nodepage := nocan.NewNodePageController()

	main.Router.GET("/api/topics", main.Topics.Index)
	main.Router.GET("/api/topics/*topic", main.Topics.Show)
	main.Router.PUT("/api/topics/*topic", main.Topics.Update)
	main.Router.GET("/api/nodes", main.Nodes.Index)
	main.Router.GET("/api/nodes/:node", main.Nodes.Show)
	main.Router.PUT("/api/nodes/:node", main.Nodes.Update)
	main.Router.GET("/api/nodes/:node/flash", main.Nodes.Firmware.Show)
	main.Router.POST("/api/nodes/:node/flash", main.Nodes.Firmware.Create)
	main.Router.GET("/api/nodes/:node/eeprom", main.Nodes.Firmware.Show)
	main.Router.POST("/api/nodes/:node/eeprom", main.Nodes.Firmware.Create)
	main.Router.GET("/api/drivers", main.Drivers.Index)
	main.Router.GET("/api/drivers/:driver", main.Drivers.Show)
	main.Router.PUT("/api/drivers/:driver", main.Drivers.Update)
	//main.Router.GET("/api/ports", main.Ports.Index)
	main.Router.ServeFiles("/static/*filepath", http.Dir("../static"))
	main.Router.GET("/nodes", nodepage.Index)
	main.Router.GET("/", homepage.Index)

	main.Run()
	fmt.Println("Done")
}

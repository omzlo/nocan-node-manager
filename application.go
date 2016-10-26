package nocan

import (
	"github.com/julienschmidt/httprouter"
)

type Application struct {
	Router *httprouter.Router
	Ports  *PortController
	Topics *TopicController
	Nodes  *NodeController
}

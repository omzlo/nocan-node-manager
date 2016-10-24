package nocan

import (
//"github.com/julienschmidt/httprouter"
//"io/ioutil"
//	"net/http"
//		"strings"
)

type PortController struct {
	Model    *PortModel
	Endpoint *CoreEndpoint
}

func NewPortController(endpoint *CoreEndpoint) *PortController {
	return &PortController{NewPortModel(), endpoint}
}

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
	Router *httprouter.Router
	Topics *TopicController
	Nodes  *NodeController
}

func NewMainTask(pm *model.PortManager) *MainTask {
	app := &MainTask{}
	//app.PortManager = model.NewPortManager()
	app.PortManager = pm
	//app.Port = pm.CreatePort("main")
	app.Router = httprouter.New()
	app.Topics = NewTopicController(app.PortManager)
	app.Nodes = NewNodeController(app.PortManager)
	return app
}

func (app *MainTask) ProcessRecv(port *model.Port) {
	m := <-port.Input

	if m.Id.IsSystem() {
		switch m.Id.GetSysFunc() {
		/*
				case NOCAN_SYS_ADDRESS_REQUEST:
					node_id, err := app.Nodes.Model.Register(m.Data)
					if err != nil {
						clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", model.UidToString(m.Data), err.Error())
					} else {
						clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", model.UidToString(m.Data), node_id)
					}
					msg := model.NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
					port.SendMessage(msg)
				case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
					// TODO
				case NOCAN_SYS_ADDRESS_LOOKUP:
					node_id, _ := app.Nodes.Model.Lookup(m.Data)
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
					port.SendMessage(msg)
			case NOCAN_SYS_TOPIC_REGISTER:
				topic_id, err := app.Topics.Model.Register(string(m.Data))
				if err != nil {
					clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to register topic %s, %s", string(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_TOPIC_REGISTER: Registered topic %s as %d", string(m.Data), topic_id)
				}
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_REGISTER_ACK, uint8(topic_id), nil)
				port.SendMessage(msg)
			case NOCAN_SYS_TOPIC_LOOKUP:
				var bitmap [8]byte
				if app.Topics.Model.Lookup(string(m.Data), bitmap[:]) {
					clog.Info("NOCAN_SYS_TOPIC_LOOKUP: Node %d succesfully found bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0, bitmap[:])
					port.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_LOOKUP: Node %d failed to find bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0xFF, nil)
					port.SendMessage(msg)
				}
			case NOCAN_SYS_TOPIC_UNREGISTER:
				var rval uint8
				if app.Topics.Model.Unregister(model.Topic(m.Id.GetSysParam())) {
					clog.Info("NOCAN_SYS_TOPIC_UNREGISTER: Node %d successfully unregistered topic %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_UNREGISTER: Node %d failed to unregister topic %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0xFF
				}
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_UNREGISTER_ACK, rval, nil)
				port.SendMessage(msg)
			case NOCAN_SYS_TOPIC_SUBSCRIBE:
				if app.Nodes.Model.Subscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d successfully subscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d failed to subscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			case NOCAN_SYS_TOPIC_UNSUBSCRIBE:
				if app.Nodes.Model.Unsubscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d successfully unsubscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d failed to unsubscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			case NOCAN_SYS_NODE_BOOT_ACK:
				// TODO:
				//default:
				//	clog.Warning("Got unknown SYS message func %d", m.Id.GetSysFunc())
		*/
		}
	}
}

func (app *MainTask) Run() {
	go http.ListenAndServe(":8888", &CheckRouter{app.Router})
	go app.Topics.Run()
	app.Nodes.Run()
	//for {
	//	app.ProcessRecv(app.Port)
	//}
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
	if r.Method == http.MethodPost && r.URL.Query().Get("_method") == http.MethodPut {
		r.Method = http.MethodPut
	}
	cr.handler.ServeHTTP(w, r)
}

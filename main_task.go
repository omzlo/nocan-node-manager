package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"strings"
)

type MainTask struct {
	TaskManager *model.TaskManager
	Router      *httprouter.Router
	Topics      *TopicController
	Nodes       *NodeController
}

func NewMainTask() *MainTask {
	app := &MainTask{}
	app.TaskManager = model.NewTaskManager()
	app.TaskManager.CreateTask("main", app)
	app.Router = httprouter.New()
	app.Topics = NewTopicController(app.TaskManager)
	app.Nodes = NewNodeController(app.TaskManager)
	return app
}

func (app *MainTask) ProcessRecv(task *model.TaskState) {
	m, _ := task.Recv()
	if m != nil {
		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_ADDRESS_REQUEST:
				node_id, err := app.Nodes.Model.Register(m.Data)
				if err != nil {
					clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", model.UidToString(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", model.UidToString(m.Data), node_id)
				}
				msg := model.NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
				task.SendMessage(msg)
			case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
				// TODO
			case NOCAN_SYS_ADDRESS_LOOKUP:
				node_id, _ := app.Nodes.Model.Lookup(m.Data)
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
				task.SendMessage(msg)
			case NOCAN_SYS_TOPIC_REGISTER:
				topic_id, err := app.Topics.Model.Register(string(m.Data))
				if err != nil {
					clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to register topic %s, %s", string(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_TOPIC_REGISTER: Registered topic %s as %d", string(m.Data), topic_id)
				}
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_REGISTER_ACK, uint8(topic_id), nil)
				task.SendMessage(msg)
			case NOCAN_SYS_TOPIC_LOOKUP:
				var bitmap [8]byte
				if app.Topics.Model.Lookup(string(m.Data), bitmap[:]) {
					clog.Info("NOCAN_SYS_TOPIC_LOOKUP: Node %d succesfully found bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0, bitmap[:])
					task.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_LOOKUP: Node %d failed to find bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0xFF, nil)
					task.SendMessage(msg)
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
				task.SendMessage(msg)
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
			default:
				clog.Warning("Got unknown SYS message func %d", m.Id.GetSysFunc())
			}
		}
	}
}

func (app *MainTask) Setup(_ *model.TaskState) {
	// empty
}

func (app *MainTask) Run(task *model.TaskState) {
	go http.ListenAndServe(":8888", &CheckRouter{app.Router})
	/*
		clock := app.Topics.Model.FindByName("/clock")
		var dummy [8]byte
		go func() {
			for {
				time.Sleep(10 * time.Second)
				//pm.SendSignal(p, SIGNAL_HEARTBEAT)
				if clock >= 0 {
					app.Publish(0, clock, dummy[:])
				}
			}
		}()
	*/
	for {
		app.ProcessRecv(task)
	}
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

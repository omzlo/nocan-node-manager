package model

import (
	"pannetrat.com/nocan/clog"
	"sync"
)

type Task int

type TaskHandler interface {
	Setup(state *TaskState)
	Run(state *TaskState)
}

const (
	TASK_STATE_CREATED = 0
	TASK_STATE_RUNNING = 1
	TASK_STATE_ZOMBIE  = 2
)

type TaskState struct {
	State   int
	Name    string
	Id      Task
	Manager *TaskManager
	Handler TaskHandler
	Input   chan *Message
	Output  chan *Message
	//Signals chan Signal
	Next *TaskState
}

func NewTaskState(name string, manager *TaskManager, handler TaskHandler) *TaskState {
	return &TaskState{
		State:   TASK_STATE_CREATED,
		Name:    name,
		Manager: manager,
		Handler: handler,
		Input:   make(chan *Message, 4),
		//Signals: make(chan Signal, 4)
	}
}

func (ts *TaskState) SendMessage(m *Message) {
	// Just to make sure a task is not deleted while we iterate through them
	ts.Manager.Mutex.Lock()
	defer ts.Manager.Mutex.Unlock()

	m.Task = ts.Id
	// clog.Debug("Sending message %d from port %d to all other ports: %s", messageCount, srcPort, m.String())
	for task := ts.Manager.Head; task != nil; task = task.Next {
		if ts != task {
			// clog.Debug("Dispatching message %d to port %d", messageCount, cindex)
			task.Input <- m
		}
	}

}

func (ts *TaskState) SendSystemMessage(node Node, fn uint8, param uint8, value []byte) {
	ts.SendMessage(NewSystemMessage(node, fn, param, value))
}

//func (ts *TaskState) WaitSystemMessage(

/*
func (ts *TaskState) SendSignal(value uint) {
	// Just to make sure a task is not deleted while we iterate through them
	ts.Manager.Mutex.Lock()
	defer ts.Manager.Mutex.Unlock()

	// clog.Debug("Sending message %d from port %d to all other ports: %s", messageCount, srcPort, m.String())
	for task := ts.Manager.Head; task != nil; task = task.Next {
		if ts != task {
			// clog.Debug("Dispatching message %d to port %d", messageCount, cindex)
			task.Signals <- CreateSignal(ts.Id, value)
		}
	}

}
*/
/*
func (ts *TaskState) Recv() *Message {
	// No need to lock, since we know the task exists.

	m := <-ts.Input:
	return m
}
*/

func (ts *TaskState) Publish(node Node, topic Topic, data []byte) {
	//clog.Debug("Publish node=%d, topic=%d dlen=%d", int(node), int(topic), len(data))
	m := NewPublishMessage(node, topic, data)
	clog.Debug("Publish %s", m.String())
	ts.SendMessage(m)
}

type TaskManager struct {
	Mutex     sync.Mutex
	LastId    Task
	TaskCount uint
	Head      *TaskState
	Zombies   chan *TaskState
}

type HandlerFunc func(state *TaskState)

func (f HandlerFunc) Setup(_ *TaskState) {
	// do nothing
}

func (f HandlerFunc) Run(state *TaskState) {
	f(state)
}

func NewTaskManager() *TaskManager {
	return &TaskManager{Zombies: make(chan *TaskState)}
}

/*
func (pm *PortModel) Each(fn func(Port, *PortState, interface{}), extra interface{}) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for iport, vport := range pm.Ports {
		fn(Port(iport), vport, extra)
	}
}
*/

func (tm *TaskManager) CreateTask(name string, handler TaskHandler) *TaskState {
	task := NewTaskState(name, tm, handler)

	tm.Mutex.Lock()

	tm.LastId++
	task.Id = tm.LastId
	task.Next = tm.Head
	tm.Head = task
	tm.TaskCount++

	tm.Mutex.Unlock()

	clog.Debug("Created task %d \"%s\"", task.Id, task.Name)
	task.Handler.Setup(task)
	return task
}

func (tm *TaskManager) CreateAndLaunchTaskFunction(name string, handler func(*TaskState)) bool {
	ts := tm.CreateTask(name, HandlerFunc(handler))
	if ts == nil {
		return false
	}
	tm.LaunchTask(ts)
	return true
}

func (tm *TaskManager) DestroyTask(state *TaskState) bool {
	var iter **TaskState

	clog.Debug("Destroying task %d \"%s\" - there are %d remaining tasks", state.Id, state.Name, tm.TaskCount-1)

	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	iter = &tm.Head

	for *iter != nil {
		if (*iter) == state {
			*iter = (*iter).Next
			tm.TaskCount--
			return true
		}
		iter = &((*iter).Next)
	}
	return false
}

func (tm *TaskManager) LaunchTask(state *TaskState) {
	go func() {
		clog.Debug("Running task %d \"%s\"", state.Id, state.Name)
		state.State = TASK_STATE_RUNNING
		state.Handler.Run(state)
		state.State = TASK_STATE_ZOMBIE
		tm.Zombies <- state
	}()
}

func (tm *TaskManager) Run() {
	// no mutex here
	for iter := tm.Head; iter != nil; iter = iter.Next {
		if iter.State == TASK_STATE_CREATED {
			tm.LaunchTask(iter)
		}
	}
	for {
		task := <-tm.Zombies
		tm.DestroyTask(task)
	}
}

package model

import (
	//"io"
	"pannetrat.com/nocan/clog"
	"sync"
	"time"
)

const (
	JobStarted   = 1
	JobCompleted = 2
	JobFailed    = 3
)

type JobState struct {
	Mutex         sync.RWMutex
	Id            uint
	Result        []byte
	Status        uint
	Progress      uint
	FailureReason error
}

func NewJob(id uint) *JobState {
	return &JobState{Id: id, Status: JobStarted, Progress: 0}
}

func (job *JobState) GetStatus() uint {
	job.Mutex.RLock()
	r := job.Status
	job.Mutex.RUnlock()
	return r
}
func (job *JobState) UpdateStatus(status uint, failureReason error) {
	job.Mutex.Lock()
	job.Status = status
	job.FailureReason = failureReason
	job.Mutex.Unlock()
}

func (job *JobState) GetProgress() uint {
	job.Mutex.RLock()
	r := job.Progress
	job.Mutex.RUnlock()
	return r
}
func (job *JobState) UpdateProgress(progress uint) {
	job.Mutex.Lock()
	job.Progress = progress
	job.Mutex.Unlock()
}

type JobModel struct {
	Mutex   sync.RWMutex
	NextId  uint
	Jobs    map[uint]*JobState
	Expired chan uint
}

func NewJobModel() *JobModel {
	return &JobModel{NextId: 0, Jobs: make(map[uint]*JobState)}
}

func (jm *JobModel) CreateJob(fn func(*JobState)) uint {
	jm.Mutex.Lock()

	jobid := jm.NextId
	job := NewJob(jobid)
	jm.Jobs[jobid] = job
	jm.NextId++

	jm.Mutex.Unlock()

	clog.Debug("Started job %d", jobid)

	go func() {
		fn(job)
		time.Sleep(time.Second * 60)
		if jm.FinalizeJob(jobid) {
			clog.Warning("Results of job %d were removed after remaining unaccessed for 60 seconds", jobid)
		}
	}()

	return jobid
}

func (jm *JobModel) FindJob(job uint) *JobState {
	jm.Mutex.RLock()
	defer jm.Mutex.RUnlock()
	return jm.Jobs[job]
}

func (jm *JobModel) FinalizeJob(job uint) bool {
	jm.Mutex.Lock()
	defer jm.Mutex.Unlock()

	if jm.Jobs[job] == nil {
		return false
	}
	clog.Debug("Terminating job %d", job)
	delete(jm.Jobs, job)
	return true
}

func (jm *JobModel) Run() {
	/* do nothing */
}

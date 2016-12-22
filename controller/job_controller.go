package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strconv"
)

type JobController struct {
	Application *Application
	Model       *model.JobModel
}

func NewJobController(app *Application) *JobController {
	controller := &JobController{Application: app, Model: model.NewJobModel()}
	return controller
}

func GetJobId(jobIdString string) (uint, error) {
	job, err := strconv.ParseUint(jobIdString, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(job), nil
}

func (jc *JobController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	jobIdString := params.ByName("id")
	jobId, err := GetJobId(jobIdString)
	if err != nil {
		view.LogHttpError(w, "Could not understand job "+jobIdString, http.StatusBadRequest)
		return
	}

	job := jc.Model.FindJob(jobId)
	if job == nil {
		view.LogHttpError(w, "Could not find job "+jobIdString, http.StatusNotFound)
		return
	}

	switch job.GetStatus() {
	case model.JobStarted:
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "%d", job.GetProgress())

	case model.JobCompleted:
		w.WriteHeader(http.StatusOK)
		if job.Result != nil {
			w.Write(job.Result)
		}
		jc.Model.FinalizeJob(jobId)

	case model.JobFailed:
		view.LogHttpError(w, fmt.Sprintf("Job %d failed, %s", jobId, job.FailureReason.Error()), http.StatusServiceUnavailable)
		jc.Model.FinalizeJob(jobId)

	}
}

func (jc *JobController) Run() {
	/* do nothing */
}

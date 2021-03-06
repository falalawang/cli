package api

import (
	"cf/models"
	"cf/net"
	"net/http"
)

type FakeAppSummaryRepo struct {
	GetSummariesInCurrentSpaceApps []models.AppSummary

	GetSummaryErrorCode string
	GetSummaryAppGuid   string
	GetSummarySummary   models.AppSummary
}

func (repo *FakeAppSummaryRepo) GetSummariesInCurrentSpace() (apps []models.AppSummary, apiResponse net.ApiResponse) {
	apps = repo.GetSummariesInCurrentSpaceApps
	return
}

func (repo *FakeAppSummaryRepo) GetSummary(appGuid string) (summary models.AppSummary, apiResponse net.ApiResponse) {
	repo.GetSummaryAppGuid = appGuid
	summary = repo.GetSummarySummary

	if repo.GetSummaryErrorCode != "" {
		apiResponse = net.NewApiResponse("Error", repo.GetSummaryErrorCode, http.StatusBadRequest)
	}

	return
}

package api

import (
	"cf/models"
	"cf/net"
)

type FakeRouteRepository struct {
	FindByHostHost  string
	FindByHostErr   bool
	FindByHostRoute models.Route

	FindByHostAndDomainHost     string
	FindByHostAndDomainDomain   string
	FindByHostAndDomainRoute    models.Route
	FindByHostAndDomainErr      bool
	FindByHostAndDomainNotFound bool

	CreatedHost       string
	CreatedDomainGuid string

	CreateInSpaceHost         string
	CreateInSpaceDomainGuid   string
	CreateInSpaceSpaceGuid    string
	CreateInSpaceCreatedRoute models.Route
	CreateInSpaceErr          bool

	BoundRouteGuid string
	BoundAppGuid   string

	UnboundRouteGuid string
	UnboundAppGuid   string

	ListErr bool
	Routes  []models.Route

	DeleteRouteGuid string
}

func (repo *FakeRouteRepository) ListRoutes(cb func(models.Route) bool) (apiResponse net.ApiResponse) {
	if repo.ListErr {
		return net.NewApiResponseWithMessage("WHOOPSIE")
	}

	for _, route := range repo.Routes {
		if !cb(route) {
			break
		}
	}
	return
}

func (repo *FakeRouteRepository) FindByHost(host string) (route models.Route, apiResponse net.ApiResponse) {
	repo.FindByHostHost = host

	if repo.FindByHostErr {
		apiResponse = net.NewApiResponseWithMessage("Route not found")
	}

	route = repo.FindByHostRoute
	return
}

func (repo *FakeRouteRepository) FindByHostAndDomain(host, domain string) (route models.Route, apiResponse net.ApiResponse) {
	repo.FindByHostAndDomainHost = host
	repo.FindByHostAndDomainDomain = domain

	if repo.FindByHostAndDomainErr {
		apiResponse = net.NewApiResponseWithMessage("Error finding Route")
	}

	if repo.FindByHostAndDomainNotFound {
		apiResponse = net.NewNotFoundApiResponse("%s %s.%s not found", "Org", host, domain)
	}

	route = repo.FindByHostAndDomainRoute
	return
}

func (repo *FakeRouteRepository) Create(host, domainGuid string) (createdRoute models.Route, apiResponse net.ApiResponse) {
	repo.CreatedHost = host
	repo.CreatedDomainGuid = domainGuid

	createdRoute.Guid = host + "-route-guid"

	return
}

func (repo *FakeRouteRepository) CreateInSpace(host, domainGuid, spaceGuid string) (createdRoute models.Route, apiResponse net.ApiResponse) {
	repo.CreateInSpaceHost = host
	repo.CreateInSpaceDomainGuid = domainGuid
	repo.CreateInSpaceSpaceGuid = spaceGuid

	if repo.CreateInSpaceErr {
		apiResponse = net.NewApiResponseWithMessage("Error")
	} else {
		createdRoute = repo.CreateInSpaceCreatedRoute
	}

	return
}

func (repo *FakeRouteRepository) Bind(routeGuid, appGuid string) (apiResponse net.ApiResponse) {
	repo.BoundRouteGuid = routeGuid
	repo.BoundAppGuid = appGuid
	return
}

func (repo *FakeRouteRepository) Unbind(routeGuid, appGuid string) (apiResponse net.ApiResponse) {
	repo.UnboundRouteGuid = routeGuid
	repo.UnboundAppGuid = appGuid
	return
}

func (repo *FakeRouteRepository) Delete(routeGuid string) (apiResponse net.ApiResponse) {
	repo.DeleteRouteGuid = routeGuid
	return
}

package api

import (
	"cf"
	"cf/configuration"
	"cf/models"
	"cf/net"
	"fmt"
	"net/url"
	"strings"
)

type ServiceRepository interface {
	PurgeServiceOffering(offering models.ServiceOffering) net.ApiResponse
	FindServiceOfferingByLabelAndProvider(name, provider string) (offering models.ServiceOffering, apiResponse net.ApiResponse)
	GetServiceOfferings() (offerings models.ServiceOfferings, apiResponse net.ApiResponse)
	FindInstanceByName(name string) (instance models.ServiceInstance, apiResponse net.ApiResponse)
	CreateServiceInstance(name, planGuid string) (identicalAlreadyExists bool, apiResponse net.ApiResponse)
	RenameService(instance models.ServiceInstance, newName string) (apiResponse net.ApiResponse)
	DeleteService(instance models.ServiceInstance) (apiResponse net.ApiResponse)
	FindServicePlanToMigrateByDescription(v1Description V1ServicePlanDescription, v2Description V2ServicePlanDescription) (v1PlanGuid, v2PlanGuid string, apiResponse net.ApiResponse)
	GetServiceInstanceCountForServicePlan(v1PlanGuid string) (count int, apiResponse net.ApiResponse)
	MigrateServicePlanFromV1ToV2(v1PlanGuid, v2PlanGuid string) net.ApiResponse
}

type CloudControllerServiceRepository struct {
	config  configuration.Reader
	gateway net.Gateway
}

func NewCloudControllerServiceRepository(config configuration.Reader, gateway net.Gateway) (repo CloudControllerServiceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceRepository) GetServiceOfferings() (offerings models.ServiceOfferings, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/services?inline-relations-depth=1", repo.config.ApiEndpoint())
	spaceGuid := repo.config.SpaceFields().Guid

	if spaceGuid != "" {
		path = fmt.Sprintf("%s/v2/spaces/%s/services?inline-relations-depth=1", repo.config.ApiEndpoint(), spaceGuid)
	}

	resources := new(PaginatedServiceOfferingResources)
	apiResponse = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)
	if apiResponse.IsNotSuccessful() {
		return
	}

	for _, r := range resources.Resources {
		offerings = append(offerings, r.ToModel())
	}

	return
}

func (repo CloudControllerServiceRepository) FindInstanceByName(name string) (instance models.ServiceInstance, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=%s&inline-relations-depth=2", repo.config.ApiEndpoint(), repo.config.SpaceFields().Guid, url.QueryEscape("name:"+name))

	resources := new(PaginatedServiceInstanceResources)
	apiResponse = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)
	if apiResponse.IsNotSuccessful() {
		return
	}

	if len(resources.Resources) == 0 {
		apiResponse = net.NewNotFoundApiResponse("Service instance '%s' not found", name)
		return
	}

	resource := resources.Resources[0]
	instance = resource.ToModel()
	return
}

func (repo CloudControllerServiceRepository) CreateServiceInstance(name, planGuid string) (identicalAlreadyExists bool, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_instances", repo.config.ApiEndpoint())
	data := fmt.Sprintf(
		`{"name":"%s","service_plan_guid":"%s","space_guid":"%s", "async": true}`,
		name, planGuid, repo.config.SpaceFields().Guid,
	)

	apiResponse = repo.gateway.CreateResource(path, repo.config.AccessToken(), strings.NewReader(data))

	if apiResponse.IsNotSuccessful() && apiResponse.ErrorCode == cf.SERVICE_INSTANCE_NAME_TAKEN {

		serviceInstance, findInstanceApiResponse := repo.FindInstanceByName(name)

		if !findInstanceApiResponse.IsNotSuccessful() &&
			serviceInstance.ServicePlan.Guid == planGuid {
			apiResponse = net.ApiResponse{}
			identicalAlreadyExists = true
			return
		}
	}
	return
}

func (repo CloudControllerServiceRepository) RenameService(instance models.ServiceInstance, newName string) (apiResponse net.ApiResponse) {
	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)

	if instance.IsUserProvided() {
		path = fmt.Sprintf("%s/v2/user_provided_service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)
	}
	return repo.gateway.UpdateResource(path, repo.config.AccessToken(), strings.NewReader(body))
}

func (repo CloudControllerServiceRepository) DeleteService(instance models.ServiceInstance) (apiResponse net.ApiResponse) {
	if len(instance.ServiceBindings) > 0 {
		return net.NewApiResponseWithMessage("Cannot delete service instance, apps are still bound to it")
	}
	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.ApiEndpoint(), instance.Guid)
	return repo.gateway.DeleteResource(path, repo.config.AccessToken())
}

func (repo CloudControllerServiceRepository) PurgeServiceOffering(offering models.ServiceOffering) net.ApiResponse {
	url := fmt.Sprintf("%s/v2/services/%s?purge=true", repo.config.ApiEndpoint(), offering.Guid)
	return repo.gateway.DeleteResource(url, repo.config.AccessToken())
}

func (repo CloudControllerServiceRepository) FindServiceOfferingByLabelAndProvider(label, provider string) (offering models.ServiceOffering, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/services?q=%s", repo.config.ApiEndpoint(), url.QueryEscape("label:"+label+";provider:"+provider))

	resources := new(PaginatedServiceOfferingResources)
	apiResponse = repo.gateway.GetResource(path, repo.config.AccessToken(), resources)

	if apiResponse.IsError() {
		return
	} else if len(resources.Resources) == 0 {
		apiResponse = net.NewNotFoundApiResponse("Service offering not found")
	} else {
		offering = resources.Resources[0].ToModel()
	}

	return
}

type V1ServicePlanDescription struct {
	ServiceName     string
	ServicePlanName string
	ServiceProvider string
}

func (v1PlanDesc V1ServicePlanDescription) String() string {
	return fmt.Sprintf("%s %s %s", v1PlanDesc.ServiceName, v1PlanDesc.ServiceProvider, v1PlanDesc.ServicePlanName)
}

type V2ServicePlanDescription struct {
	ServiceName     string
	ServicePlanName string
}

func (v2PlanDesc V2ServicePlanDescription) String() string {
	return fmt.Sprintf("%s %s", v2PlanDesc.ServiceName, v2PlanDesc.ServicePlanName)
}

func (repo CloudControllerServiceRepository) FindServicePlanToMigrateByDescription(v1Description V1ServicePlanDescription, v2Description V2ServicePlanDescription) (v1PlanGuid, v2PlanGuid string, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_plans?inline-relations-depth=1", repo.config.ApiEndpoint())

	response := new(PaginatedServicePlanResources)
	apiResponse = repo.gateway.GetResource(path, repo.config.AccessToken(), response)
	if apiResponse.IsNotSuccessful() {
		return
	}

	for _, resource := range response.Resources {
		if v1PlanGuid == "" {
			serviceOffering := resource.Entity.ServiceOffering.Entity

			matchingPlan := resource.Entity.Name == v1Description.ServicePlanName
			matchingService := serviceOffering.Label == v1Description.ServiceName
			matchingProvider := serviceOffering.Provider == v1Description.ServiceProvider
			if matchingPlan && matchingService && matchingProvider {
				v1PlanGuid = resource.Metadata.Guid
			}
		}

		if v2PlanGuid == "" {
			serviceOffering := resource.Entity.ServiceOffering.Entity

			matchingPlan := resource.Entity.Name == v2Description.ServicePlanName
			matchingService := serviceOffering.Label == v2Description.ServiceName
			matchingProvider := serviceOffering.Provider == ""
			if matchingPlan && matchingService && matchingProvider {
				v2PlanGuid = resource.Metadata.Guid
			}
		}
	}

	if v1PlanGuid == "" {
		apiResponse = net.NewNotFoundApiResponse("Plan %s cannot be found", v1Description)
		return
	}

	if v2PlanGuid == "" {
		apiResponse = net.NewNotFoundApiResponse("Plan %s cannot be found", v2Description)
		return
	}

	return
}

func (repo CloudControllerServiceRepository) MigrateServicePlanFromV1ToV2(v1PlanGuid, v2PlanGuid string) (apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_plans/%s/service_instances", repo.config.ApiEndpoint(), v1PlanGuid)
	body := strings.NewReader(fmt.Sprintf(`{"service_plan_guid":"%s"}`, v2PlanGuid))
	request, apiResponse := repo.gateway.NewRequest("PUT", path, repo.config.AccessToken(), body)
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) GetServiceInstanceCountForServicePlan(v1PlanGuid string) (count int, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_plans/%s/service_instances?results-per-page=1", repo.config.ApiEndpoint(), v1PlanGuid)
	response := new(PaginatedServiceInstanceResources)
	apiResponse = repo.gateway.GetResource(path, repo.config.AccessToken(), response)
	count = response.TotalResults
	return
}

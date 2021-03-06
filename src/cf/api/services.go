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

type PaginatedServiceOfferingResources struct {
	Resources []ServiceOfferingResource
}

type ServiceOfferingResource struct {
	Metadata Metadata
	Entity   ServiceOfferingEntity
}

func (resource ServiceOfferingResource) ToFields() (fields models.ServiceOfferingFields) {
	fields.Label = resource.Entity.Label
	fields.Version = resource.Entity.Version
	fields.Provider = resource.Entity.Provider
	fields.Description = resource.Entity.Description
	fields.Guid = resource.Metadata.Guid
	fields.DocumentationUrl = resource.Entity.DocumentationUrl
	return
}

func (resource ServiceOfferingResource) ToModel() (offering models.ServiceOffering) {
	offering.ServiceOfferingFields = resource.ToFields()
	for _, p := range resource.Entity.ServicePlans {
		servicePlan := models.ServicePlanFields{}
		servicePlan.Name = p.Entity.Name
		servicePlan.Guid = p.Metadata.Guid
		offering.Plans = append(offering.Plans, servicePlan)
	}
	return offering
}

type ServiceOfferingEntity struct {
	Label            string
	Version          string
	Description      string
	DocumentationUrl string `json:"documentation_url"`
	Provider         string
	ServicePlans     []ServicePlanResource `json:"service_plans"`
}

type ServicePlanResource struct {
	Metadata Metadata
	Entity   ServicePlanEntity
}

func (resource ServicePlanResource) ToFields() (fields models.ServicePlanFields) {
	fields.Guid = resource.Metadata.Guid
	fields.Name = resource.Entity.Name
	return
}

type ServicePlanEntity struct {
	Name            string
	ServiceOffering ServiceOfferingResource `json:"service"`
}

type PaginatedServiceInstanceResources struct {
	Resources []ServiceInstanceResource
}

type ServiceInstanceResource struct {
	Metadata Metadata
	Entity   ServiceInstanceEntity
}

func (resource ServiceInstanceResource) ToFields() (fields models.ServiceInstanceFields) {
	fields.Guid = resource.Metadata.Guid
	fields.Name = resource.Entity.Name
	return
}

func (resource ServiceInstanceResource) ToModel() (instance models.ServiceInstance) {
	instance.ServiceInstanceFields = resource.ToFields()
	instance.ServicePlan = resource.Entity.ServicePlan.ToFields()
	instance.ServiceOffering = resource.Entity.ServicePlan.Entity.ServiceOffering.ToFields()

	instance.ServiceBindings = []models.ServiceBindingFields{}
	for _, bindingResource := range resource.Entity.ServiceBindings {
		instance.ServiceBindings = append(instance.ServiceBindings, bindingResource.ToFields())
	}
	return
}

type ServiceInstanceEntity struct {
	Name            string
	ServiceBindings []ServiceBindingResource `json:"service_bindings"`
	ServicePlan     ServicePlanResource      `json:"service_plan"`
}

type ServiceBindingResource struct {
	Metadata Metadata
	Entity   ServiceBindingEntity
}

func (resource ServiceBindingResource) ToFields() (fields models.ServiceBindingFields) {
	fields.Url = resource.Metadata.Url
	fields.Guid = resource.Metadata.Guid
	fields.AppGuid = resource.Entity.AppGuid
	return
}

type ServiceBindingEntity struct {
	AppGuid string `json:"app_guid"`
}

type ServiceRepository interface {
	PurgeServiceOffering(offering models.ServiceOffering) net.ApiResponse
	FindServiceOfferingByLabelAndProvider(name, provider string) (offering models.ServiceOffering, apiResponse net.ApiResponse)
	GetServiceOfferings() (offerings models.ServiceOfferings, apiResponse net.ApiResponse)
	FindInstanceByName(name string) (instance models.ServiceInstance, apiResponse net.ApiResponse)
	CreateServiceInstance(name, planGuid string) (identicalAlreadyExists bool, apiResponse net.ApiResponse)
	RenameService(instance models.ServiceInstance, newName string) (apiResponse net.ApiResponse)
	DeleteService(instance models.ServiceInstance) (apiResponse net.ApiResponse)
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

package commands_test

import (
	. "cf/commands"
	"cf/configuration"
	"cf/models"
	"cf/net"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strconv"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	"testhelpers/maker"
	testterm "testhelpers/terminal"
)

var _ = Describe("Testing with ginkgo", func() {
	It("TestSuccessfullyLoggingInWithNumericalPrompts", func() {
		c := setUpLoginTestContext()

		OUT_OF_RANGE_CHOICE := "3"

		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password", OUT_OF_RANGE_CHOICE, "2", OUT_OF_RANGE_CHOICE, "1"}

		org1 := models.Organization{}
		org1.Guid = "some-org-guid"
		org1.Name = "some-org"

		org2 := models.Organization{}
		org2.Guid = "my-org-guid"
		org2.Name = "my-org"

		space1 := models.Space{}
		space1.Guid = "my-space-guid"
		space1.Name = "my-space"

		space2 := models.Space{}
		space2.Guid = "some-space-guid"
		space2.Name = "some-space"

		c.orgRepo.Organizations = []models.Organization{org1, org2}
		c.spaceRepo.Spaces = []models.Space{space1, space2}

		callLogin(c)

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Select an org"},
			{"1. some-org"},
			{"2. my-org"},
			{"Select a space"},
			{"1. my-space"},
			{"2. some-space"},
		})

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.orgRepo.FindByNameName).To(Equal("my-org"))
		Expect(c.spaceRepo.FindByNameName).To(Equal("my-space"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithStringPrompts", func() {
		c := setUpLoginTestContext()

		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password", "my-org", "my-space"}

		org1 := models.Organization{}
		org1.Guid = "some-org-guid"
		org1.Name = "some-org"

		org2 := models.Organization{}
		org2.Guid = "my-org-guid"
		org2.Name = "my-org"

		space1 := models.Space{}
		space1.Guid = "my-space-guid"
		space1.Name = "my-space"

		space2 := models.Space{}
		space2.Guid = "some-space-guid"
		space2.Name = "some-space"

		c.orgRepo.Organizations = []models.Organization{org1, org2}
		c.spaceRepo.Spaces = []models.Space{space1, space2}

		callLogin(c)

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Select an org"},
			{"1. some-org"},
			{"2. my-org"},
			{"Select a space"},
			{"1. my-space"},
			{"2. some-space"},
		})

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.orgRepo.FindByNameName).To(Equal("my-org"))
		Expect(c.spaceRepo.FindByNameName).To(Equal("my-space"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestLoggingInWithTooManyOrgsDoesNotShowOrgList", func() {
		c := setUpLoginTestContext()

		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password", "my-org-1", "my-space"}

		for i := 0; i < 60; i++ {
			id := strconv.Itoa(i)
			org := models.Organization{}
			org.Guid = "my-org-guid-" + id
			org.Name = "my-org-" + id
			c.orgRepo.Organizations = append(c.orgRepo.Organizations, org)
		}

		c.orgRepo.FindByNameOrganization = c.orgRepo.Organizations[1]

		space1 := models.Space{}
		space1.Guid = "my-space-guid"
		space1.Name = "my-space"

		space2 := models.Space{}
		space2.Guid = "some-space-guid"
		space2.Name = "some-space"

		c.spaceRepo.Spaces = []models.Space{space1, space2}

		callLogin(c)

		testassert.SliceDoesNotContain(c.ui.Outputs, testassert.Lines{
			{"my-org-2"},
		})
		Expect(c.orgRepo.FindByNameName).To(Equal("my-org-1"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid-1"))
	})

	It("TestSuccessfullyLoggingInWithFlags", func() {
		c := setUpLoginTestContext()

		c.Flags = []string{"-a", "api.example.com", "-u", "user@example.com", "-p", "password", "-o", "my-org", "-s", "my-space"}

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithEndpointSetInConfig", func() {
		c := setUpLoginTestContext()

		c.Flags = []string{"-o", "my-org", "-s", "my-space"}
		c.ui.Inputs = []string{"user@example.com", "password"}
		c.Config.SetApiEndpoint("http://api.example.com")

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("http://api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("http://api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithOrgSetInConfig", func() {
		c := setUpLoginTestContext()

		org := models.OrganizationFields{}
		org.Name = "my-org"
		org.Guid = "my-org-guid"
		c.Config.SetOrganizationFields(org)

		c.Flags = []string{"-s", "my-space"}
		c.ui.Inputs = []string{"http://api.example.com", "user@example.com", "password"}

		c.orgRepo.FindByNameOrganization = models.Organization{}

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("http://api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("http://api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithOrgAndSpaceSetInConfig", func() {
		c := setUpLoginTestContext()

		org := models.OrganizationFields{}
		org.Name = "my-org"
		org.Guid = "my-org-guid"
		c.Config.SetOrganizationFields(org)

		space := models.SpaceFields{}
		space.Guid = "my-space-guid"
		space.Name = "my-space"
		c.Config.SetSpaceFields(space)

		c.ui.Inputs = []string{"http://api.example.com", "user@example.com", "password"}

		c.orgRepo.FindByNameOrganization = models.Organization{}
		c.spaceRepo.FindByNameInOrgSpace = models.Space{}

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("http://api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("http://api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithOnlyOneOrg", func() {
		c := setUpLoginTestContext()

		org := models.Organization{}
		org.Name = "my-org"
		org.Guid = "my-org-guid"

		c.Flags = []string{"-s", "my-space"}
		c.ui.Inputs = []string{"http://api.example.com", "user@example.com", "password"}
		c.orgRepo.FindByNameOrganization = models.Organization{}
		c.orgRepo.Organizations = []models.Organization{org}

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("http://api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("http://api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestSuccessfullyLoggingInWithOnlyOneSpace", func() {
		c := setUpLoginTestContext()

		space := models.Space{}
		space.Guid = "my-space-guid"
		space.Name = "my-space"

		c.Flags = []string{"-o", "my-org"}
		c.ui.Inputs = []string{"http://api.example.com", "user@example.com", "password"}
		c.spaceRepo.Spaces = []models.Space{space}

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("http://api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal("my-space-guid"))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		Expect(c.endpointRepo.UpdateEndpointReceived).To(Equal("http://api.example.com"))
		Expect(c.authRepo.Email).To(Equal("user@example.com"))
		Expect(c.authRepo.Password).To(Equal("password"))

		Expect(c.ui.ShowConfigurationCalled).To(BeTrue())
	})

	It("TestUnsuccessfullyLoggingInWithAuthError", func() {
		c := setUpLoginTestContext()

		c.Flags = []string{"-u", "user@example.com"}
		c.ui.Inputs = []string{"api.example.com", "password", "password2", "password3"}
		c.authRepo.AuthError = true

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(BeEmpty())
		Expect(c.Config.SpaceFields().Guid).To(BeEmpty())
		Expect(c.Config.AccessToken()).To(BeEmpty())
		Expect(c.Config.RefreshToken()).To(BeEmpty())

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Failed"},
		})
	})

	It("TestUnsuccessfullyLoggingInWithUpdateEndpointError", func() {
		c := setUpLoginTestContext()

		c.ui.Inputs = []string{"api.example.com"}
		c.endpointRepo.UpdateEndpointError = net.NewApiResponseWithMessage("Server error")

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(BeEmpty())
		Expect(c.Config.OrganizationFields().Guid).To(BeEmpty())
		Expect(c.Config.SpaceFields().Guid).To(BeEmpty())
		Expect(c.Config.AccessToken()).To(BeEmpty())
		Expect(c.Config.RefreshToken()).To(BeEmpty())

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Failed"},
		})
	})

	It("TestUnsuccessfullyLoggingInWithOrgFindByNameErr", func() {
		c := setUpLoginTestContext()

		c.Flags = []string{"-u", "user@example.com", "-o", "my-org", "-s", "my-space"}
		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password"}
		c.orgRepo.FindByNameErr = true

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(BeEmpty())
		Expect(c.Config.SpaceFields().Guid).To(BeEmpty())
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Failed"},
		})
	})

	It("TestUnsuccessfullyLoggingInWithSpaceFindByNameErr", func() {
		c := setUpLoginTestContext()

		c.Flags = []string{"-u", "user@example.com", "-o", "my-org", "-s", "my-space"}
		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password"}
		c.spaceRepo.FindByNameErr = true

		callLogin(c)

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("my-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(BeEmpty())
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Failed"},
		})
	})

	It("TestSuccessfullyLoggingInWithoutTargetOrg", func() {
		c := setUpLoginTestContext()

		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password", ""}

		org1 := maker.NewOrg(maker.Overrides{"name": "org1"})
		org2 := maker.NewOrg(maker.Overrides{"name": "org2"})
		c.orgRepo.Organizations = []models.Organization{org1, org2}

		callLogin(c)

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Select an org (or press enter to skip):"},
		})
		testassert.SliceDoesNotContain(c.ui.Outputs, testassert.Lines{
			{"Select a space", "or press enter to skip"},
		})
		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal(""))
		Expect(c.Config.SpaceFields().Guid).To(Equal(""))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))
	})

	It("TestSuccessfullyLoggingInWithoutTargetSpace", func() {
		c := setUpLoginTestContext()

		c.ui.Inputs = []string{"api.example.com", "user@example.com", "password", ""}

		org := models.Organization{}
		org.Guid = "some-org-guid"
		org.Name = "some-org"

		space1 := maker.NewSpace(maker.Overrides{"name": "some-space", "guid": "some-space-guid"})
		space2 := maker.NewSpace(maker.Overrides{"name": "other-space", "guid": "other-space-guid"})

		c.orgRepo.Organizations = []models.Organization{org}
		c.spaceRepo.Spaces = []models.Space{space1, space2}

		callLogin(c)

		testassert.SliceContains(c.ui.Outputs, testassert.Lines{
			{"Select a space (or press enter to skip):"},
		})
		testassert.SliceDoesNotContain(c.ui.Outputs, testassert.Lines{
			{"FAILED"},
		})

		Expect(c.Config.ApiEndpoint()).To(Equal("api.example.com"))
		Expect(c.Config.OrganizationFields().Guid).To(Equal("some-org-guid"))
		Expect(c.Config.SpaceFields().Guid).To(Equal(""))
		Expect(c.Config.AccessToken()).To(Equal("my_access_token"))
		Expect(c.Config.RefreshToken()).To(Equal("my_refresh_token"))
	})
})

type LoginTestContext struct {
	Flags  []string
	Config configuration.ReadWriter

	ui           *testterm.FakeUI
	authRepo     *testapi.FakeAuthenticationRepository
	endpointRepo *testapi.FakeEndpointRepo
	orgRepo      *testapi.FakeOrgRepository
	spaceRepo    *testapi.FakeSpaceRepository
}

func setUpLoginTestContext() (c *LoginTestContext) {
	c = new(LoginTestContext)
	c.Config = testconfig.NewRepository()

	c.ui = &testterm.FakeUI{}

	c.authRepo = &testapi.FakeAuthenticationRepository{
		AccessToken:  "my_access_token",
		RefreshToken: "my_refresh_token",
		Config:       c.Config,
	}
	c.endpointRepo = &testapi.FakeEndpointRepo{Config: c.Config}

	org := models.Organization{}
	org.Name = "my-org"
	org.Guid = "my-org-guid"

	c.orgRepo = &testapi.FakeOrgRepository{
		Organizations: []models.Organization{org},
	}

	space := models.Space{}
	space.Name = "my-space"
	space.Guid = "my-space-guid"

	c.spaceRepo = &testapi.FakeSpaceRepository{
		Spaces: []models.Space{space},
	}

	return
}

func callLogin(c *LoginTestContext) {
	l := NewLogin(c.ui, c.Config, c.authRepo, c.endpointRepo, c.orgRepo, c.spaceRepo)
	testcmd.RunCommand(l, testcmd.NewContext("login", c.Flags), nil)
}

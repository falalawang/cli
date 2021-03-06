package application_test

import (
	. "cf/commands/application"
	"cf/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
)

var _ = Describe("Testing with ginkgo", func() {
	It("TestRenameAppFailsWithUsage", func() {
		reqFactory := &testreq.FakeReqFactory{}
		appRepo := &testapi.FakeApplicationRepository{}

		ui := callRename([]string{}, reqFactory, appRepo)
		Expect(ui.FailedWithUsage).To(BeTrue())

		ui = callRename([]string{"foo"}, reqFactory, appRepo)
		Expect(ui.FailedWithUsage).To(BeTrue())
	})
	It("TestRenameRequirements", func() {

		appRepo := &testapi.FakeApplicationRepository{}

		reqFactory := &testreq.FakeReqFactory{LoginSuccess: true}
		callRename([]string{"my-app", "my-new-app"}, reqFactory, appRepo)
		Expect(testcmd.CommandDidPassRequirements).To(BeTrue())
		Expect(reqFactory.ApplicationName).To(Equal("my-app"))
	})
	It("TestRenameRun", func() {

		appRepo := &testapi.FakeApplicationRepository{}
		app := models.Application{}
		app.Name = "my-app"
		app.Guid = "my-app-guid"
		reqFactory := &testreq.FakeReqFactory{LoginSuccess: true, Application: app}
		ui := callRename([]string{"my-app", "my-new-app"}, reqFactory, appRepo)

		Expect(appRepo.UpdateAppGuid).To(Equal(app.Guid))
		Expect(*appRepo.UpdateParams.Name).To(Equal("my-new-app"))
		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Renaming app", "my-app", "my-new-app", "my-org", "my-space", "my-user"},
			{"OK"},
		})
	})
})

func callRename(args []string, reqFactory *testreq.FakeReqFactory, appRepo *testapi.FakeApplicationRepository) (ui *testterm.FakeUI) {
	ui = new(testterm.FakeUI)
	ctxt := testcmd.NewContext("rename", args)

	configRepo := testconfig.NewRepositoryWithDefaults()
	cmd := NewRenameApp(ui, configRepo, appRepo)
	testcmd.RunCommand(cmd, ctxt, reqFactory)
	return
}

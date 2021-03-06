package service_test

import (
	"cf/api"
	. "cf/commands/service"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
)

func callCreateUserProvidedService(args []string, inputs []string, userProvidedServiceInstanceRepo api.UserProvidedServiceInstanceRepository) (fakeUI *testterm.FakeUI) {
	fakeUI = &testterm.FakeUI{Inputs: inputs}
	ctxt := testcmd.NewContext("create-user-provided-service", args)
	reqFactory := &testreq.FakeReqFactory{}

	config := testconfig.NewRepositoryWithDefaults()

	cmd := NewCreateUserProvidedService(fakeUI, config, userProvidedServiceInstanceRepo)
	testcmd.RunCommand(cmd, ctxt, reqFactory)
	return
}

var _ = Describe("Testing with ginkgo", func() {
	It("TestCreateUserProvidedServiceWithParameterList", func() {
		repo := &testapi.FakeUserProvidedServiceInstanceRepo{}
		ui := callCreateUserProvidedService([]string{"-p", `"foo, bar, baz"`, "my-custom-service"},
			[]string{"foo value", "bar value", "baz value"},
			repo,
		)

		testassert.SliceContains(ui.Prompts, testassert.Lines{
			{"foo"},
			{"bar"},
			{"baz"},
		})

		Expect(repo.CreateName).To(Equal("my-custom-service"))
		Expect(repo.CreateParams).To(Equal(map[string]string{
			"foo": "foo value",
			"bar": "bar value",
			"baz": "baz value",
		}))

		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Creating user provided service", "my-custom-service", "my-org", "my-space", "my-user"},
			{"OK"},
		})
	})

	It("TestCreateUserProvidedServiceWithJson", func() {
		repo := &testapi.FakeUserProvidedServiceInstanceRepo{}
		ui := callCreateUserProvidedService([]string{"-p", `{"foo": "foo value", "bar": "bar value", "baz": "baz value"}`, "my-custom-service"},
			[]string{},
			repo,
		)

		Expect(ui.Prompts).To(BeEmpty())

		Expect(repo.CreateName).To(Equal("my-custom-service"))
		Expect(repo.CreateParams).To(Equal(map[string]string{
			"foo": "foo value",
			"bar": "bar value",
			"baz": "baz value",
		}))

		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Creating user provided service"},
			{"OK"},
		})
	})
	It("TestCreateUserProvidedServiceWithNoSecondArgument", func() {

		userProvidedServiceInstanceRepo := &testapi.FakeUserProvidedServiceInstanceRepo{}
		ui := callCreateUserProvidedService([]string{"my-custom-service"},
			[]string{},
			userProvidedServiceInstanceRepo,
		)

		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Creating user provided service"},
			{"OK"},
		})
	})
	It("TestCreateUserProvidedServiceWithSyslogDrain", func() {

		repo := &testapi.FakeUserProvidedServiceInstanceRepo{}

		ui := callCreateUserProvidedService([]string{"-l", "syslog://example.com", "-p", `{"foo": "foo value", "bar": "bar value", "baz": "baz value"}`, "my-custom-service"},
			[]string{},
			repo,
		)
		Expect(repo.CreateDrainUrl).To(Equal("syslog://example.com"))
		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Creating user provided service"},
			{"OK"},
		})
	})
})

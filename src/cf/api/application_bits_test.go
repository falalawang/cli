package api_test

import (
	"archive/zip"
	"cf"
	. "cf/api"
	"cf/models"
	"cf/net"
	"fileutils"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	mr "github.com/tjarratt/mr_t"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	testapi "testhelpers/api"
	testconfig "testhelpers/configuration"
	testnet "testhelpers/net"
	"time"
)

var permissionsToSet os.FileMode
var expectedPermissionBits os.FileMode

func init() {
	permissionsToSet = 0467
	fileutils.TempFile("permissionedFile", func(file *os.File, err error) {
		if err != nil {
			panic("could not create tmp file")
		}

		fileInfo, err := file.Stat()
		if err != nil {
			panic("could not stat tmp file")
		}

		expectedPermissionBits = fileInfo.Mode()
		if runtime.GOOS != "windows" {
			expectedPermissionBits |= permissionsToSet & 0111
		}
	})
}

var _ = Describe("ApplicationBitsRepository", func() {
	Describe("ApplicationBitsRepository", func() {
		It("TestUploadWithInvalidDirectory", func() {
			config := testconfig.NewRepository()
			gateway := net.NewCloudControllerGateway()
			zipper := &cf.ApplicationZipper{}

			repo := NewCloudControllerApplicationBitsRepository(config, gateway, zipper)

			apiResponse := repo.UploadApp("app-guid", "/foo/bar", func(path string, uploadSize, fileCount uint64) {})
			Expect(apiResponse.IsNotSuccessful()).To(BeTrue())
			assert.Contains(mr.T(), apiResponse.Message, filepath.Join("foo", "bar"))
		})
		It("TestUploadApp", func() {

			dir, err := os.Getwd()
			assert.NoError(mr.T(), err)
			dir = filepath.Join(dir, "../../fixtures/example-app")
			err = os.Chmod(filepath.Join(dir, "Gemfile"), permissionsToSet)

			assert.NoError(mr.T(), err)

			_, apiResponse := callUploadApp(dir, defaultRequests)
			Expect(apiResponse.IsSuccessful()).To(BeTrue())
		})
		It("TestCreateUploadDirWithAZipFile", func() {

			dir, err := os.Getwd()
			assert.NoError(mr.T(), err)
			dir = filepath.Join(dir, "../../fixtures/example-app.zip")

			_, apiResponse := callUploadApp(dir, defaultRequests)
			Expect(apiResponse.IsSuccessful()).To(BeTrue())
		})
		It("TestCreateUploadDirWithAZipLikeFile", func() {

			dir, err := os.Getwd()
			assert.NoError(mr.T(), err)
			dir = filepath.Join(dir, "../../fixtures/example-app.azip")

			_, apiResponse := callUploadApp(dir, defaultRequests)
			Expect(apiResponse.IsSuccessful()).To(BeTrue())
		})
		It("TestUploadAppFailsWhilePushingBits", func() {

			dir, err := os.Getwd()
			assert.NoError(mr.T(), err)
			dir = filepath.Join(dir, "../../fixtures/example-app")

			requests := []testnet.TestRequest{
				uploadApplicationRequest,
				createProgressEndpoint("running"),
				createProgressEndpoint("failed"),
			}
			_, apiResponse := callUploadApp(dir, requests)
			Expect(apiResponse.IsSuccessful()).To(BeFalse())
		})
	})
})

var uploadApplicationRequest = testapi.NewCloudControllerTestRequest(testnet.TestRequest{
	Method:  "PUT",
	Path:    "/v2/apps/my-cool-app-guid/bits",
	Matcher: uploadBodyMatcher,
	Response: testnet.TestResponse{
		Status: http.StatusCreated,
		Body: `
{
	"metadata":{
		"guid": "my-job-guid",
		"url": "/v2/jobs/my-job-guid"
	}
}
	`},
})
var defaultRequests = []testnet.TestRequest{
	uploadApplicationRequest,
	createProgressEndpoint("running"),
	createProgressEndpoint("finished"),
}

var expectedApplicationContent = []string{"Gemfile", "Gemfile.lock", "manifest.yml", "app.rb", "config.ru"}

var uploadBodyMatcher = func(t mr.TestingT, request *http.Request) {
	err := request.ParseMultipartForm(4096)
	if err != nil {
		assert.Fail(t, "Failed parsing multipart form", err)
		return
	}
	defer request.MultipartForm.RemoveAll()

	Expect(len(request.MultipartForm.Value)).To(Equal(1), "Should have 1 value")
	valuePart, ok := request.MultipartForm.Value["resources"]
	assert.True(t, ok, "Resource manifest not present")
	Expect(len(valuePart)).To(Equal(1), "Wrong number of values")

	resourceManifest := valuePart[0]
	chompedResourceManifest := strings.Replace(resourceManifest, "\n", "", -1)
	Expect(chompedResourceManifest).To(Equal("[]"), "Resources do not match")

	Expect(len(request.MultipartForm.File)).To(Equal(1), "Wrong number of files")

	fileHeaders, ok := request.MultipartForm.File["application"]
	assert.True(t, ok, "Application file part not present")
	Expect(len(fileHeaders)).To(Equal(1), "Wrong number of files")

	applicationFile := fileHeaders[0]
	Expect(applicationFile.Filename).To(Equal("application.zip"), "Wrong file name")

	file, err := applicationFile.Open()
	if err != nil {
		assert.Fail(t, "Cannot get multipart file", err.Error())
		return
	}

	length, err := strconv.ParseInt(applicationFile.Header.Get("content-length"), 10, 64)
	if err != nil {
		assert.Fail(t, "Cannot convert content-length to int", err.Error())
		return
	}

	zipReader, err := zip.NewReader(file, length)
	if err != nil {
		assert.Fail(t, "Error reading zip content", err.Error())
		return
	}

	Expect(len(zipReader.File)).To(Equal(5), "Wrong number of files in zip")
	Expect(zipReader.File[0].Mode()).To(Equal(expectedPermissionBits))

nextFile:
	for _, f := range zipReader.File {
		for _, expected := range expectedApplicationContent {
			if f.Name == expected {
				continue nextFile
			}
		}
		assert.Fail(t, "Missing file: "+f.Name)
	}
}

func createProgressEndpoint(status string) (req testnet.TestRequest) {
	body := fmt.Sprintf(`
	{
		"entity":{
			"status":"%s"
		}
	}`, status)

	req.Method = "GET"
	req.Path = "/v2/jobs/my-job-guid"
	req.Response = testnet.TestResponse{
		Status: http.StatusCreated,
		Body:   body,
	}

	return
}

func callUploadApp(dir string, requests []testnet.TestRequest) (app models.Application, apiResponse net.ApiResponse) {
	ts, handler := testnet.NewTLSServer(GinkgoT(), requests)
	defer ts.Close()

	configRepo := testconfig.NewRepositoryWithDefaults()
	configRepo.SetApiEndpoint(ts.URL)
	gateway := net.NewCloudControllerGateway()
	gateway.PollingThrottle = time.Duration(0)
	zipper := cf.ApplicationZipper{}
	repo := NewCloudControllerApplicationBitsRepository(configRepo, gateway, zipper)

	var (
		reportedPath                          string
		reportedFileCount, reportedUploadSize uint64
	)
	apiResponse = repo.UploadApp("my-cool-app-guid", dir, func(path string, uploadSize, fileCount uint64) {
		reportedPath = path
		reportedUploadSize = uploadSize
		reportedFileCount = fileCount
	})

	Expect(reportedPath).To(Equal(dir))
	Expect(reportedFileCount).To(Equal(uint64(len(expectedApplicationContent))))
	Expect(reportedUploadSize).To(Equal(uint64(1094)))
	Expect(handler.AllRequestsCalled()).To(BeTrue())
	return
}

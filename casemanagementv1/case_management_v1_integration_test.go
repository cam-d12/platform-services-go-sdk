//go:build integration
// +build integration

/**
 * (C) Copyright IBM Corp. 2020.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package casemanagementv1_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/casemanagementv1"
	common "github.com/IBM/platform-services-go-sdk/common"
)

var _ = Describe("Case Management - Integration Tests", func() {
	const externalConfigFile = "../case_management.env"

	var (
		service *casemanagementv1.CaseManagementV1
		err     error

		config map[string]string

		caseNumber   string
		commentValue = "Test comment"

		// Configured resource CRN to use in tests.
		resourceCRN string

		// Model instances needed by the tests.
		resourcePayload  []casemanagementv1.ResourcePayload
		watchlistPayload *casemanagementv1.Watchlist
	)

	var shouldSkipTest = func() {
		Skip("External configuration is not available, skipping...")
	}

	It("Successfully load the configuration", func() {
		_, err = os.Stat(externalConfigFile)
		if err != nil {
			Skip("External configuration file not found, skipping tests: " + err.Error())
		}

		os.Setenv("IBM_CREDENTIALS_FILE", externalConfigFile)
		config, err = core.GetServiceProperties(casemanagementv1.DefaultServiceName)
		if err != nil {
			Skip("Error loading service properties, skipping tests: " + err.Error())
		}
		serviceURL := config["URL"]
		if serviceURL == "" {
			Skip("Unable to load service URL configuration property, skipping tests")
		}

		resourceCRN = config["RESOURCE_CRN"]
		Expect(resourceCRN).ToNot(BeEmpty())

		shouldSkipTest = func() {}

		// Initialize required model instances.
		resourcePayload = []casemanagementv1.ResourcePayload{casemanagementv1.ResourcePayload{
			CRN: &resourceCRN,
		}}

		watchlistPayload = &casemanagementv1.Watchlist{
			Watchlist: []casemanagementv1.User{
				casemanagementv1.User{
					Realm:  core.StringPtr("IBMid"),
					UserID: core.StringPtr("abc@ibm.com"),
				},
			},
		}
	})

	It(`Successfully created CaseManagementV1 service instance`, func() {
		shouldSkipTest()

		service, err = casemanagementv1.NewCaseManagementV1UsingExternalConfig(
			&casemanagementv1.CaseManagementV1Options{},
		)

		Expect(err).To(BeNil())
		Expect(service).ToNot(BeNil())

		goLogger := log.New(GinkgoWriter, "", log.LstdFlags)
		core.SetLogger(core.NewLogger(core.LevelError, goLogger, goLogger))
		service.EnableRetries(4, 30*time.Second)

		// Set client timeout.
		service.Service.Client.Timeout = 2 * time.Minute

		fmt.Fprintf(GinkgoWriter, "\nService URL: %s\n", service.Service.GetServiceURL())
	})

	Describe("Create a case", func() {
		var options *casemanagementv1.CreateCaseOptions
		BeforeEach(func() {
			offeringType := &casemanagementv1.OfferingType{
				Group: core.StringPtr(casemanagementv1.OfferingTypeGroupCRNServiceNameConst),
				Key:   core.StringPtr("cloud-object-storage"),
			}

			offeringPayload := &casemanagementv1.Offering{
				Name: core.StringPtr("Cloud Object Storage"),
				Type: offeringType,
			}

			options = service.NewCreateCaseOptions("technical", "Test case for Go SDK", "Test case for Go SDK")
			options.SetSeverity(4)
			options.SetOffering(offeringPayload)
		})

		It("Successfully created a technical case", func() {
			shouldSkipTest()

			ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelFunc()

			result, detailedResponse, err := service.CreateCaseWithContext(ctx, options)
			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "CreateCase() result:\n%s\n", common.ToJSON(result))
			Expect(*result.Number).To(Not(BeNil()))
			Expect(*result.ShortDescription).To(Equal(*options.Subject))
			Expect(*result.Description).To(Equal(*options.Description))
			Expect(int64(*result.Severity)).To(Equal(*options.Severity))

			caseNumber = *result.Number

		})

		It("Bad payload used to create a case", func() {
			shouldSkipTest()
			options.SetType("invalid_type")
			options.Severity = nil
			options.Offering = nil

			ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelFunc()

			_, detailedResponse, err := service.CreateCaseWithContext(ctx, options)
			Expect(err).To(Not(BeNil()))
			Expect(detailedResponse.StatusCode).To(Not(Equal(200)))
		})
	})

	Describe("Get Cases", func() {
		var options *casemanagementv1.GetCasesOptions

		BeforeEach(func() {
			options = service.NewGetCasesOptions()
		})

		It("Successfully got cases with default params", func() {
			shouldSkipTest()

			result, detailedResponse, err := service.GetCases(options)
			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "GetCases(default params) result:\n%s\n", common.ToJSON(result))
			Expect(*result.TotalCount).To(Not(BeNil()))
			Expect(*result.First).To(Not(BeNil()))
			Expect(*result.Next).To(Not(BeNil()))
			Expect(*result.Last).To(Not(BeNil()))
			Expect(result.Cases).To(Not(BeNil()))
		})

		It("Successfully got cases with non-default params", func() {
			shouldSkipTest()

			options.SetOffset(10)
			options.SetLimit(20)
			options.SetFields([]string{
				casemanagementv1.GetCasesOptionsFieldsNumberConst,
				casemanagementv1.GetCasesOptionsFieldsCommentsConst,
				casemanagementv1.GetCasesOptionsFieldsCreatedAtConst,
			})

			result, detailedResponse, err := service.GetCases(options)
			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "GetCases(non-default params) result:\n%s\n", common.ToJSON(result))
			Expect(*result.TotalCount).To(Not(BeNil()))
			Expect(*result.First).To(Not(BeNil()))
			Expect(*result.Next).To(Not(BeNil()))
			Expect(*result.Last).To(Not(BeNil()))
			Expect(result.Cases).To(Not(BeNil()))

			testCase := result.Cases[0]
			Expect(testCase).To(Not(BeNil()))
			Expect(testCase.Number).To(Not(BeNil()))
			Expect(testCase.Comments).To(Not(BeNil()))
			Expect(testCase.CreatedAt).To(Not(BeNil()))

			// extra properties should be excluded in the response
			Expect(testCase.Severity).To(BeNil())
			Expect(testCase.Contact).To(BeNil())
		})

		It("Failed to get cases with bad params", func() {
			shouldSkipTest()

			options.SetFields([]string{"invalid_fields"})

			_, detailedResponse, err := service.GetCases(options)
			Expect(err).To(Not(BeNil()))
			Expect(detailedResponse.StatusCode).To(Not(Equal(200)))
		})
	})

	Describe("Get a specific case", func() {
		var options *casemanagementv1.GetCaseOptions

		BeforeEach(func() {
			options = service.NewGetCaseOptions(caseNumber)
		})

		It("Successfully got a case with default params", func() {
			shouldSkipTest()

			result, detailedResponse, err := service.GetCase(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "GetCase(default) result:\n%s\n", common.ToJSON(result))
			Expect(*result.Number).To(Equal(caseNumber))
		})

		It("Successfully got a case with field filtering", func() {
			shouldSkipTest()

			options.SetFields([]string{"number", "severity"})
			result, detailedResponse, err := service.GetCase(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "GetCase(field filtering) result:\n%s\n", common.ToJSON(result))
			Expect(*result.Number).To(Equal(caseNumber))
			Expect(result.Severity).To(Not(BeNil()))
			Expect(result.Contact).To(BeNil())
		})

		It("Failed to get a case with bad params", func() {
			shouldSkipTest()

			options.SetFields([]string{"invalid_field"})
			_, detailedResponse, err := service.GetCase(options)
			Expect(err).To(Not(BeNil()))
			Expect(detailedResponse.StatusCode).To(Not(Equal(200)))
		})
	})

	Describe("Add comment", func() {
		var options *casemanagementv1.AddCommentOptions

		BeforeEach(func() {
			options = service.NewAddCommentOptions(caseNumber, commentValue)
		})

		It("Successfully added a comment to a case", func() {
			shouldSkipTest()
			result, detailedResponse, err := service.AddComment(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "AddComment() result:\n%s\n", common.ToJSON(result))
			Expect(*result.Value).To(Equal(commentValue))
			Expect(result.AddedAt).To(Not(BeNil()))
			Expect(result.AddedBy).To(Not(BeNil()))
		})
	})

	Describe("Add watchlist", func() {
		var options *casemanagementv1.AddWatchlistOptions

		BeforeEach(func() {
			options = service.NewAddWatchlistOptions(caseNumber)
			options.SetWatchlist(watchlistPayload.Watchlist)
		})

		It("Successfully added users to case watchlist", func() {
			shouldSkipTest()
			result, detailedResponse, err := service.AddWatchlist(options)

			Expect(err).To((BeNil()))
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "AddWatchlist() result:\n%s\n", common.ToJSON(result))

			// We expect the call to fail because the fake user is not associated with the account.
			Expect(len(result.Failed)).To(Equal(len(watchlistPayload.Watchlist)))
		})
	})

	Describe("Remove watchlist", func() {
		var options *casemanagementv1.RemoveWatchlistOptions
		BeforeEach(func() {
			options = service.NewRemoveWatchlistOptions(caseNumber)
			options.SetWatchlist(watchlistPayload.Watchlist)
		})

		It("Successfully removed users from case watchlist", func() {
			shouldSkipTest()
			result, detailedResponse, err := service.RemoveWatchlist(options)

			Expect(err).To((BeNil()))
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "RemoveWatchlist() result:\n%s\n", common.ToJSON(result))
		})
	})

	Describe("Update status", func() {

		It("Succefully resolve a case", func() {
			shouldSkipTest()
			resolvePayload, _ := service.NewResolvePayload(casemanagementv1.ResolvePayloadActionResolveConst, 1)
			options := service.NewUpdateCaseStatusOptions(caseNumber, resolvePayload)

			result, detailedResponse, err := service.UpdateCaseStatus(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "UpdateCaseStatus(resolve) result:\n%s\n", common.ToJSON(result))
			Expect(*result.Status).To(Equal("Resolved"))
		})

		It("Succefully unresolve a case", func() {
			shouldSkipTest()
			unresolvePayload, _ := service.NewUnresolvePayload(casemanagementv1.UnresolvePayloadActionUnresolveConst, "Test unresolve")
			options := service.NewUpdateCaseStatusOptions(caseNumber, unresolvePayload)

			result, detailedResponse, err := service.UpdateCaseStatus(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "UpdateCaseStatus(unresolve) result:\n%s\n", common.ToJSON(result))
			Expect(*result.Status).To(Equal("In Progress"))
		})
	})

	Describe("Modify attachments", func() {
		var fileID string

		It("Successfully uploaded file", func() {
			shouldSkipTest()
			fileInput, _ := service.NewFileWithMetadata(io.NopCloser(strings.NewReader("hello world")))
			fileInput.Filename = core.StringPtr("GO SDK test file.png")
			fileInput.ContentType = core.StringPtr("application/octet-stream")

			filePayload := []casemanagementv1.FileWithMetadata{*fileInput}
			options := service.NewUploadFileOptions(caseNumber, filePayload)

			result, detailedResponse, err := service.UploadFile(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To(Equal(200))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "UploadFile() result:\n%s\n", common.ToJSON(result))
			Expect(*result.ID).To(Not(BeNil()))
			Expect(*result.Filename).To(Equal(*fileInput.Filename))

			// store file id so that we could remove it in the next test
			fileID = *result.ID
		})

		It("Successfully deleted file", func() {
			shouldSkipTest()

			if fileID == "" {
				Skip("Case does not have target file to remove. Skipping ....")
			}

			options := service.NewDeleteFileOptions(caseNumber, fileID)

			_, detailedResponse, err := service.DeleteFile(options)

			Expect(err).To((BeNil()))
			Expect(detailedResponse.StatusCode).To((Equal(200)))
		})
	})

	Describe("Add Resource", func() {
		It("Successfully added a resource", func() {
			shouldSkipTest()
			crn := *resourcePayload[0].CRN
			options := service.NewAddResourceOptions(caseNumber)
			options.SetCRN(crn)

			result, detailedResponse, err := service.AddResource(options)

			Expect(err).To(BeNil())
			Expect(detailedResponse.StatusCode).To((Equal(200)))
			Expect(result).ToNot(BeNil())
			fmt.Fprintf(GinkgoWriter, "AddResource() result:\n%s\n", common.ToJSON(result))
			Expect(*result.CRN).To(Equal(crn))
		})
	})
})

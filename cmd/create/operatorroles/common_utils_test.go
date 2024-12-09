package operatorroles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Create dns domain", func() {
	var ctrl *gomock.Controller
	var runtime *rosa.Runtime

	var testPartition = "test"
	var testArn = "arn:aws:iam::123456789012:role/test"
	var testVersion = "2012-10-17"
	var mockClient *aws.MockClient

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		runtime = rosa.NewRuntime()
		mockClient = aws.NewMockClient(ctrl)
		runtime.AWSClient = mockClient
		mockClient.EXPECT().GetCreator().Return(&aws.Creator{Partition: testPartition}, nil)

		mockClient.EXPECT().IsPolicyExists(gomock.Any()).Return(nil, nil).AnyTimes()

		creator, err := runtime.AWSClient.GetCreator()
		Expect(err).ToNot(HaveOccurred())
		runtime.Creator = creator
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Common Utils for create/operatorroles Test", func() {
		When("getHcpSharedVpcPolicy", func() {
			It("OK: Gets policy arn back", func() {
				returnedArn := "arn:aws:iam::123123123123:policy/test"
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return(returnedArn, nil)
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(arn).To(Equal(returnedArn))
			})
			It("KO: Returns empty policy when fails", func() {
				mockClient.EXPECT().EnsurePolicy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any()).Return("", errors.UserErrorf("Failed"))
				arn, err := getHcpSharedVpcPolicy(runtime, testArn, testVersion)
				Expect(err).To(HaveOccurred())
				Expect(arn).To(Equal(""))
			})
		})
	})
})

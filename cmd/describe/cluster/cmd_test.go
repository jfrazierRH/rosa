package cluster

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	version string = "4.10.1"
	state   string = "running"
)

var (
	now                             = time.Now()
	expectEmptyCuster               = []byte(`{"displayName":"displayname","kind":"Cluster"}`)
	expectClusterWithNameAndIDValue = []byte(
		`{"displayName":"displayname","id":"bar","kind":"Cluster","name":"foo"}`)
	expectClusterWithExternalAuthConfig = []byte(
		`{"displayName":"displayname","external_auth_config":{"enabled":true},"kind":"Cluster"}`)
	expectClusterWithEtcd = []byte(
		`{"aws":{"etcd_encryption":{"kms_key_arn":"arn:aws:kms:us-west-2:125374464689:key/` +
			`41fccc11-b089-test-aeff-test"}},"displayName":"displayname","etcd_encryption":true,"kind":"Cluster"}`)
	expectClusterWithAap = []byte(
		`{"aws":{"additional_allowed_principals":["foobar"]},"displayName":"displayname","kind":"Cluster"}`)
	expectClusterWithNameAndValueAndUpgradeInformation = []byte(
		`{"displayName":"displayname","id":"bar","kind":"Cluster","name":"foo","scheduledUpgrade":{"nextRun":"` +
			now.Format("2006-01-02 15:04 MST") + `","state":"` + state + `","version":"` +
			version + `"}}`)
	expectEmptyClusterWithNameAndValueAndUpgradeInformation = []byte(
		`{"displayName":"displayname","kind":"Cluster","scheduledUpgrade":{"nextRun":"` +
			now.Format("2006-01-02 15:04 MST") + `","state":"` +
			state + `","version":"` +
			version + `"}}`)
	clusterWithNameAndID, emptyCluster, clusterWithExternalAuthConfig, clusterWithAap, clusterWithKms *cmv1.Cluster
	emptyUpgradePolicy, upgradePolicyWithVersionAndNextRun                                            *cmv1.UpgradePolicy
	//nolint
	emptyUpgradeState, upgradePolicyWithState *cmv1.UpgradePolicyState

	berr error
)
var _ = BeforeSuite(func() {
	clusterWithNameAndID, berr = cmv1.NewCluster().Name("foo").ID("bar").Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyCluster, berr = cmv1.NewCluster().Build()
	Expect(berr).NotTo(HaveOccurred())
	externalAuthConfig := cmv1.NewExternalAuthConfig().Enabled(true)
	clusterWithExternalAuthConfig, berr = cmv1.NewCluster().ExternalAuthConfig(externalAuthConfig).Build()
	Expect(berr).NotTo(HaveOccurred())
	additionalAllowedPrincipals := cmv1.NewAWS().AdditionalAllowedPrincipals("foobar")
	clusterWithAap, berr = cmv1.NewCluster().AWS(additionalAllowedPrincipals).Build()
	Expect(berr).NotTo(HaveOccurred())
	clusterWithKms, berr = cmv1.NewCluster().EtcdEncryption(true).AWS(cmv1.NewAWS().
		EtcdEncryption(cmv1.NewAwsEtcdEncryption().KMSKeyARN(
			"arn:aws:kms:us-west-2:125374464689:key/41fccc11-b089-test-aeff-test"))).
		Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyUpgradePolicy, berr = cmv1.NewUpgradePolicy().Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyUpgradeState, berr = cmv1.NewUpgradePolicyState().Build()
	Expect(berr).NotTo(HaveOccurred())
	upgradePolicyWithVersionAndNextRun, berr = cmv1.NewUpgradePolicy().Version(version).NextRun(now).Build()
	Expect(berr).NotTo(HaveOccurred())
	upgradePolicyWithState, berr = cmv1.NewUpgradePolicyState().Value(cmv1.UpgradePolicyStateValue(state)).Build()
	Expect(berr).NotTo(HaveOccurred())

})
var _ = Describe("Cluster description", Ordered, func() {

	Context("when displaying clusters with output json", func() {

		DescribeTable("When displaying clusters with output json",
			printJson,
			Entry("Prints empty when all values are empty",
				func() *cmv1.Cluster { return emptyCluster },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectEmptyCuster, nil),

			Entry("Prints cluster information only",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster and upgrade information",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return upgradePolicyWithState },
				expectClusterWithNameAndValueAndUpgradeInformation, nil),

			Entry("Prints empty cluster with cluster information",
				func() *cmv1.Cluster { return emptyCluster },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return upgradePolicyWithState },
				expectEmptyClusterWithNameAndValueAndUpgradeInformation, nil),

			Entry("Prints cluster information only when no upgrade policy state is found",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when no upgrade policy version and next run is found",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when upgrade policy is nil",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return nil },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when upgrade policy state is nil",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information with the external authentication provider",
				func() *cmv1.Cluster { return clusterWithExternalAuthConfig },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithExternalAuthConfig, nil),

			Entry("Prints cluster information with the additional allowed principals",
				func() *cmv1.Cluster { return clusterWithAap },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithAap, nil),

			Entry("Prints cluster information with KMS ARN",
				func() *cmv1.Cluster { return clusterWithKms },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithEtcd, nil),
		)
	})
})

var _ = Describe("getClusterRegistryConfig", func() {
	It("Should return expected output", func() {
		mockCa := make(map[string]string)
		mockCa["registry.io"] = "-----BEGIN CERTIFICATE-----\nlalala\n-----END CERTIFICATE-----\n"
		mockCa["registry.io2"] = "-----BEGIN CERTIFICATE-----\nlalala\n-----END CERTIFICATE-----\n"
		mockCluster, err := cmv1.NewCluster().RegistryConfig(cmv1.NewClusterRegistryConfig().AdditionalTrustedCa(mockCa).
			RegistrySources(cmv1.NewRegistrySources().
				AllowedRegistries([]string{"allow1.com", "allow2.com"}...).
				InsecureRegistries([]string{"insecure1.com", "insecure2.com"}...).
				BlockedRegistries([]string{"block1.com", "block2.com"}...)).
			AllowedRegistriesForImport(cmv1.NewRegistryLocation().
				DomainName("quay.io").Insecure(true)).
			PlatformAllowlist(cmv1.NewRegistryAllowlist().ID("test-id"))).Build()
		Expect(err).NotTo(HaveOccurred())

		mockAllowlist, err := cmv1.NewRegistryAllowlist().ID("test-id").
			Registries([]string{"registry1.io", "registry2.io"}...).Build()
		Expect(err).NotTo(HaveOccurred())
		output := getClusterRegistryConfig(mockCluster, mockAllowlist)
		expectedOutput := " - Allowed Registries:      allow1.com,allow2.com\n" +
			" - Blocked Registries:      block1.com,block2.com\n" +
			" - Insecure Registries:     insecure1.com,insecure2.com\n" +
			" - Allowed Registries for Import:         \n" +
			"    - Domain Name:          quay.io\n" +
			"    - Insecure:             true\n" +
			" - Platform Allowlist:      test-id\n" +
			"    - Registries:           registry1.io,registry2.io\n" +
			" - Additional Trusted CA:         \n" +
			"    - registry.io: REDACTED\n" +
			"    - registry.io2: REDACTED\n"
		Expect(output).To(Equal(expectedOutput))
	})
})

var _ = Describe("getZeroEgressStatus", func() {
	var (
		ssoServer, apiServer *ghttp.Server
		r                    *rosa.Runtime

		// GET /api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress
		zeroEgressResp = `
	{
      "kind":"ProductTechnologyPreview",
      "id":"hcp-zero-egress",
      "href":"/api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress",
      "end_date":"2024-11-16T00:00:00Z",
      "additional_text":"Zero Egress"
	}`
	)

	BeforeEach(func() {
		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()
		apiServer.SetAllowUnhandledRequests(true)
		apiServer.SetUnhandledRequestStatusCode(http.StatusInternalServerError)

		// Create the token:
		accessToken := MakeTokenString("Bearer", 15*time.Minute)

		// Prepare the server:
		ssoServer.AppendHandlers(
			RespondWithAccessToken(accessToken),
		)
		logger, err := logging.NewGoLoggerBuilder().
			Debug(false).
			Build()
		Expect(err).ToNot(HaveOccurred())
		// Set up the connection with the fake config
		connection, err := sdk.NewConnectionBuilder().
			Logger(logger).
			Tokens(accessToken).
			URL(apiServer.URL()).
			Build()
		// Initialize client object
		Expect(err).To(BeNil())
		ocmClient := ocm.NewClientWithConnection(connection)

		r = rosa.NewRuntime()
		r.OCMClient = ocmClient
		r.Creator = &aws.Creator{
			ARN:       "fake",
			AccountID: "123",
			IsSTS:     false,
		}
	})

	// TODO: Temporarily disabled due to bug in CS
	XIt("Should include zero egress enabled in the output", func() {
		// GET /api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				zeroEgressResp,
			),
		)

		mockCluster, err := cmv1.NewCluster().Properties(map[string]string{"zero_egress": "true"}).Build()
		Expect(err).NotTo(HaveOccurred())
		output, err := getZeroEgressStatus(r, mockCluster)
		Expect(err).NotTo(HaveOccurred())
		expectedOutput := "Zero Egress:                Enabled\n"
		Expect(output).To(Equal(expectedOutput))
	})
	// TODO: Temporarily disabled due to bug in CS
	XIt("Should include zero egress disabled in the output", func() {
		// GET /api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				zeroEgressResp,
			),
		)

		mockCluster, err := cmv1.NewCluster().Properties(map[string]string{"zero_egress": "false"}).Build()
		Expect(err).NotTo(HaveOccurred())
		output, err := getZeroEgressStatus(r, mockCluster)
		Expect(err).NotTo(HaveOccurred())
		expectedOutput := "Zero Egress:                Disabled\n"
		Expect(output).To(Equal(expectedOutput))
	})
	It("Should not include zero egress in the output", func() {
		// GET /api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusNotFound, "",
			),
		)

		mockCluster, err := cmv1.NewCluster().Properties(map[string]string{"zero_egress": "true"}).Build()
		Expect(err).NotTo(HaveOccurred())
		output, err := getZeroEgressStatus(r, mockCluster)
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(Equal(""))
	})
	It("Should not include zero egress in the output", func() {
		// GET /api/clusters_mgmt/v1/products/rosa/technology_previews/hcp-zero-egress
		apiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusInternalServerError, "",
			),
		)

		mockCluster, err := cmv1.NewCluster().Properties(map[string]string{"zero_egress": "true"}).Build()
		Expect(err).NotTo(HaveOccurred())
		_, err = getZeroEgressStatus(r, mockCluster)
		Expect(err).To(HaveOccurred())
	})
})

func printJson(cluster func() *cmv1.Cluster,
	upgrade func() *cmv1.UpgradePolicy,
	state func() *cmv1.UpgradePolicyState,
	expected []byte,
	err error) {
	f, er := formatCluster(cluster(), upgrade(), state(), "displayname")
	if err != nil {
		Expect(er).To(Equal(err))
	}
	Expect(er).To(BeNil())
	v, er := json.Marshal(f)
	Expect(er).NotTo(HaveOccurred())
	Expect(v).To(Equal(expected))
}

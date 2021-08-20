package v1_api_tests

import (
	"fmt"
	"testing"

	v3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
	"github.com/rancher/rancher/tests/go-validation/aws"
	"github.com/rancher/rancher/tests/go-validation/environmentvariables"

	"github.com/rancher/rancher/tests/go-validation/clients"
	"github.com/rancher/rancher/tests/go-validation/cluster"
	"github.com/rancher/rancher/tests/go-validation/namegenerator"
	"github.com/rancher/rancher/tests/go-validation/tokenregistration"
	"github.com/rancher/rancher/tests/go-validation/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func CreateEC2Instances(s *ProvisioningTestSuite) {
	s.T().Log("Creating EC2 Instances")
	s.baseEC2Name = "automation-"
	s.randString = namegenerator.RandStringLowerBytes(5)
	ec2NodeName := s.baseEC2Name + s.randString

	newClient, err := aws.NewEC2Client()
	s.ec2Client = newClient
	require.NoError(s.T(), err)

	s.nodes, err = s.ec2Client.CreateNodes(ec2NodeName, true, s.nodeConfig.NumNodes)
	require.NoError(s.T(), err)
	s.T().Log("Successfully created EC2 Instances")
}

func GenerateClusterName(s *ProvisioningTestSuite) string {
	s.T().Log("Create Cluster")
	s.baseClusterName = "customcluster-" + s.randString + "-"
	//randomize name
	clusterName := s.baseClusterName + namegenerator.RandStringLowerBytes(5)
	return clusterName
}

func ProvisionClusters(s *ProvisioningTestSuite) {
	CreateEC2Instances(s)

	s.T().Logf("User is %s", s.bearerName)

	clusterName := GenerateClusterName(s)
	s.clusterNames = append(s.clusterNames, clusterName)

	provisioningClient, err := clients.NewProvisioningClient(s.bearerToken)
	require.NoError(s.T(), err)

	clusterConfig := cluster.NewRKE2ClusterConfig(clusterName, s.clusterNamespace, cluster.CNI, "", cluster.KubernetesVersion, nil)

	clusterObj := cluster.NewCluster(s.clusterNamespace, provisioningClient)
	s.clusters = append(s.clusters, clusterObj)

	s.T().Logf("Creating Cluster %s", clusterName)
	v1Cluster, err := clusterObj.CreateCluster(clusterConfig)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), v1Cluster.Name, clusterName)

	s.T().Logf("Created Cluster %s", v1Cluster.Name)

	getCluster, err := clusterObj.PollCluster(v1Cluster.Name)
	require.NoError(s.T(), err)

	s.T().Log("new client for token registration")

	managementClient, err := clients.NewManagementClient(s.bearerToken)
	s.managementClient = managementClient
	require.NoError(s.T(), err)

	s.T().Log("Client creation was successful")

	clusterRegistrationToken := tokenregistration.NewClusterRegistrationToken(s.managementClient)

	// all registration creation and existence.
	// time.Sleep(10 * time.Second)

	s.T().Log("Before getting registration token")
	token, err := clusterRegistrationToken.GetRegistrationToken(getCluster.Status.ClusterName)
	require.NoError(s.T(), err)
	s.T().Log("Successfully got registration token")

	for key, node := range s.nodes {
		s.T().Logf("Execute Registration Command for node %s", node.NodeID)
		command := fmt.Sprintf("%s %s", token.InsecureNodeCommand, s.nodeConfig.Roles[key])

		err = node.ExecuteCommand(command)
		require.NoError(s.T(), err)
	}

	s.T().Logf("Checking status of cluster %s", v1Cluster.ClusterName)
	//check cluster status
	ready, err := clusterObj.CheckClusterStatus(clusterName)
	assert.NoError(s.T(), err)
	assert.True(s.T(), ready)
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ProvisioningTestSuite struct {
	suite.Suite

	// server config
	host         string
	password     string
	adminToken   string
	userToken    string
	cni          string
	k8sVersion   string
	rolesPerNode string

	// AWS environment variables
	awsInstanceType    string
	awsRegion          string
	awsRegionAZ        string
	awsAMI             string
	awsSecurityGroup   string
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSSHKeyName      string
	awsCICDInstanceTag string
	awsIAMProfile      string
	awsUser            string
	awsVolumeSize      int
	sshPath            string

	ec2Client        *aws.EC2Client
	nodes            []*aws.EC2Node
	nodeConfig       *cluster.NodeConfig
	nodeRoles        []string
	baseEC2Name      string
	baseClusterName  string
	clusterNamespace string
	randString       string

	bearerName       string
	bearerToken      string
	managementClient *v3.Client

	clusterNames []string
	clusters     []*cluster.Cluster

	rancherCleanup bool
}

// The SetupSuite method will be run by testify once, at the very
// start of the testing suite, before any tests are run.
func (s *ProvisioningTestSuite) SetupSuite() {
	/* Can further abstract this type of config by feeding values in via Flags, config files, or environment variables
	 as detailed here: https://dev.to/ilyakaznacheev/a-clean-way-to-pass-configs-in-a-go-application-1g64
	 prospective tools:
						- https://github.com/BurntSushi/toml
	 					- gopkg.in/yaml.v2
						- https://dev.to/ilyakaznacheev/a-clean-way-to-pass-configs-in-a-go-application-1g64
	 					- https://github.com/ilyakaznacheev/cleanenv
						- https://github.com/peterbourgon/ff

	Passing --timeout dynamically per-test may be desired once average test durations are determined in order to avoid
	false negatives when setting the --timeout too low
	*/

	s.host = clients.Host
	s.password = user.Password
	s.adminToken = clients.AdminToken
	s.userToken = clients.UserToken
	s.cni = cluster.CNI
	s.k8sVersion = cluster.KubernetesVersion
	s.clusterNamespace = "fleet-default"
	s.rancherCleanup = environmentvariables.RancherCleanup()
	s.awsInstanceType = aws.AWSInstanceType
	s.awsRegion = aws.AWSRegion
	s.awsRegionAZ = aws.AWSRegionAZ
	s.awsAMI = aws.AWSAMI
	s.awsSecurityGroup = aws.AWSSecurityGroup
	s.awsAccessKeyID = aws.AWSAccessKeyID
	s.awsSecretAccessKey = aws.AWSSecretAccessKey
	s.awsSSHKeyName = aws.AWSSSHKeyName
	s.awsCICDInstanceTag = aws.AWSCICDInstanceTag
	s.awsIAMProfile = aws.AWSIAMProfile
	s.awsUser = aws.AWSUser
	s.awsVolumeSize = aws.AWSVolumeSize
	s.sshPath = aws.SSHPath
}

// Do stuff before each test
func (s *ProvisioningTestSuite) SetupTest() {
	s.T().Log("Setup test")
}

func (s *ProvisioningTestSuite) BeforeTest(suiteName, testName string) {
	s.T().Log("before test")
}

// All methods that begin with "Test" are run as tests within a
// suite.
func (s *ProvisioningTestSuite) TestRKE2CustomCluster_CreateUser() {
	userName := "automationuser"
	displayName := "Automation User"
	mustChangePassword := false
	s.nodeRoles = []string{
		"--etcd",
		"--controlplane",
		"--worker",
	}

	s.nodeConfig = cluster.NewNodeConfig("3 nodes - 1 role per node", 3, s.nodeRoles)
	for s.bearerName, s.bearerToken = range clients.BearerTokensList() {
		ProvisionClusters(s)
	}

	newUser := user.NewUser(s.managementClient)
	v3User, err := newUser.CreateUser(userName, s.password, displayName, mustChangePassword)
	require.NoError(s.T(), err)
	s.T().Logf("Successfully Created v3User: %s", v3User.ID)

	expected_id, err := newUser.GetUser(v3User.ID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), v3User.ID, expected_id)
}

func (s *ProvisioningTestSuite) TestRKE2CustomCluster_TableDriven() {
	roles0 := []string{
		"--etcd --controlplane --worker",
	}

	roles1 := []string{
		"--etcd",
		"--controlplane",
		"--worker",
	}

	tests := []cluster.NodeConfig{
		{Name: "1 Node all roles", NumNodes: 1, Roles: roles0},
		{Name: "3 nodes - 1 role per node", NumNodes: 3, Roles: roles1},
	}

	/* An initial implementation of table tests with suites. An ideal implementation would allow a setup + teardown
	   function to be run before and after each sub-test, respectively.
	*/
	for _, tt := range tests {
		// SetupTableTest(s)

		for s.bearerName, s.bearerToken = range clients.BearerTokensList() {
			s.nodeConfig = cluster.NewNodeConfig(tt.Name+" "+s.bearerName, tt.NumNodes, tt.Roles)
			s.Run(s.nodeConfig.Name, func() {
				ProvisionClusters(s)
			})
		}

		// TeardownTableTest(s)
	}
}

func (s *ProvisioningTestSuite) AfterTest(suiteName, testName string) {
	s.T().Log("After test")
}

// Do stuff after each test
func (s *ProvisioningTestSuite) TearDownTest() {
	s.T().Log("Teardown test")
}

// Do stuff after the suite
func (s *ProvisioningTestSuite) TearDownSuite() {
	if s.rancherCleanup {
		err := cluster.ClusterCleanup(s.ec2Client, s.clusters, s.clusterNames, s.nodes)
		require.NoError(s.T(), err)
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(ProvisioningTestSuite))
}

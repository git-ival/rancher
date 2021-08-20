package v1_api_tests

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	apisV1 "github.com/rancher/rancher/pkg/apis/provisioning.cattle.io/v1"
	coreV1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/rancher/rancher/tests/go-validation/aws"

	"github.com/rancher/rancher/tests/go-validation/clients"
	"github.com/rancher/rancher/tests/go-validation/cloudcredentials"
	"github.com/rancher/rancher/tests/go-validation/cluster"
	"github.com/rancher/rancher/tests/go-validation/environmentvariables"
	"github.com/rancher/rancher/tests/go-validation/machinepool"
	"github.com/rancher/rancher/tests/go-validation/namegenerator"
	"github.com/rancher/rancher/tests/go-validation/tokenregistration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	digitalOceanCloudCredentialName = "docloudcredential"
	namespace                       = "fleet-default"
	defaultRandStringLength         = 5
	baseDOClusterName               = "docluster"
	baseCustomClusterName           = "customcluster"
	baseEC2Name                     = "rancherautomation"
	defaultTokenName                = "default-token"
)

var nodesAndRoles = os.Getenv("NODE_ROLES")

func TestProvisioning_RKE2CustomCluster(t *testing.T) {
	roles0 := []string{
		"--etcd --controlplane --worker",
	}

	roles1 := []string{
		"--etcd",
		"--controlplane",
		"--worker",
	}

	tests := []struct {
		name     string
		numNodes int64
		roles    []string
	}{
		{"1 Node all roles", 1, roles0},
		{"3 nodes - 1 role per node", 3, roles1},
	}
	var clusterNames []string
	var clusters []*cluster.Cluster

	for _, tt := range tests {
		for userName, bearerToken := range clients.BearerTokensList() {
			name := tt.name + " " + userName
			t.Run(name, func(t *testing.T) {
				t.Logf("User is %s", userName)

				newClient, err := aws.NewEC2Client()
				require.NoError(t, err)

				ec2NodeName := baseEC2Name + namegenerator.RandStringLowerBytes(defaultRandStringLength)
				nodes, err := newClient.CreateNodes(ec2NodeName, true, tt.numNodes)
				require.NoError(t, err)
				t.Log("Successfully created EC2 Instances")

				t.Log("Create Cluster")
				//randomize name
				clusterName := baseCustomClusterName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
				clusterNames = append(clusterNames, clusterName)

				provisioningClient, err := clients.NewProvisioningClient(bearerToken)
				require.NoError(t, err)

				clusterConfig := cluster.NewRKE2ClusterConfig(clusterName, namespace, cluster.CNI, "", cluster.KubernetesVersion, nil)

				clusterObj := cluster.NewCluster(namespace, provisioningClient)
				clusters = append(clusters, clusterObj)

				t.Logf("Creating Cluster %s", clusterName)
				v1Cluster, err := clusterObj.CreateCluster(clusterConfig)
				require.NoError(t, err)

				assert.Equal(t, v1Cluster.Name, clusterName)

				t.Logf("Created Cluster %s", v1Cluster.ClusterName)

				getCluster, err := clusterObj.PollCluster(v1Cluster.Name)
				require.NoError(t, err)

				t.Log("new client for token registration")

				managementClient, err := clients.NewManagementClient(bearerToken)
				require.NoError(t, err)

				t.Log("Client creation was successful")

				clusterTokenRegistration := tokenregistration.NewClusterRegistrationToken(managementClient)

				// all registration creation and existence.
				// time.Sleep(10 * time.Second)

				t.Log("Before getting registration token")
				token, err := clusterTokenRegistration.GetRegistrationToken(getCluster.Status.ClusterName)
				require.NoError(t, err)
				t.Log("Successfully got registration token")

				for key, node := range nodes {
					t.Logf("Execute Registration Command for node %s", node.NodeID)
					command := fmt.Sprintf("%s %s", token.InsecureNodeCommand, tt.roles[key])

					err = node.ExecuteCommand(command)
					require.NoError(t, err)
				}

				t.Logf("Checking status of cluster %s", v1Cluster.ClusterName)
				//check cluster status
				ready, err := clusterObj.CheckClusterStatus(clusterName)
				assert.NoError(t, err)
				assert.True(t, ready)

				if environmentvariables.RancherCleanup() {
					err := cluster.ClusterCleanup(newClient, clusters, clusterNames, nodes)
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestProvisioning_RKE2CustomClusterDynamicInput(t *testing.T) {
	if nodesAndRoles == "" {
		t.Skip()
	}

	for userName, bearerToken := range clients.BearerTokensList() {
		t.Run(userName, func(t *testing.T) {
			rolesSlice := strings.Split(nodesAndRoles, "|")
			numNodes := len(rolesSlice)

			var clusterNames []string
			var clusters []*cluster.Cluster

			newClient, err := aws.NewEC2Client()
			require.NoError(t, err)

			t.Log("Creating EC2 Instances")
			ec2NodeName := baseEC2Name + namegenerator.RandStringLowerBytes(defaultRandStringLength)
			nodes, err := newClient.CreateNodes(ec2NodeName, true, int64(numNodes))
			require.NoError(t, err)
			t.Log("Successfully created EC2 Instances")

			t.Log("Create Cluster")
			//randomize name
			clusterName := baseCustomClusterName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
			clusterNames = append(clusterNames, clusterName)

			provisioningClient, err := clients.NewProvisioningClient(bearerToken)
			require.NoError(t, err)

			clusterConfig := cluster.NewRKE2ClusterConfig(clusterName, namespace, cluster.CNI, "", cluster.KubernetesVersion, nil)

			clusterObj := cluster.NewCluster(namespace, provisioningClient)
			clusters = append(clusters, clusterObj)

			t.Logf("Creating Cluster %s", clusterName)
			v1Cluster, err := clusterObj.CreateCluster(clusterConfig)
			require.NoError(t, err)

			assert.Equal(t, v1Cluster.Name, clusterName)

			t.Logf("Created Cluster %s", v1Cluster.ClusterName)

			getCluster, err := clusterObj.PollCluster(v1Cluster.Name)
			require.NoError(t, err)

			t.Log("new client for token registration")

			managementClient, err := clients.NewManagementClient(bearerToken)
			require.NoError(t, err)

			clusterTokenRegistration := tokenregistration.NewClusterRegistrationToken(managementClient)

			token, err := clusterTokenRegistration.GetRegistrationToken(getCluster.Status.ClusterName)
			require.NoError(t, err)

			for key, node := range nodes {
				t.Logf("Execute Registration Command for node %s", node.NodeID)
				roles := rolesSlice[key]
				roleCommands := strings.Split(roles, ",")
				var finalRoleCommand string
				for _, roleCommand := range roleCommands {
					finalRoleCommand += fmt.Sprintf(" --%s", roleCommand)
				}

				command := fmt.Sprintf("%s %s", token.InsecureNodeCommand, finalRoleCommand)

				err = node.ExecuteCommand(command)
				require.NoError(t, err)
			}

			t.Logf("Checking status of cluster %s", v1Cluster.ClusterName)
			//check cluster status
			ready, err := clusterObj.CheckClusterStatus(clusterName)
			assert.NoError(t, err)
			assert.True(t, ready)

			if environmentvariables.RancherCleanup() {
				err := cluster.ClusterCleanup(newClient, clusters, clusterNames, nodes)
				require.NoError(t, err)
			}
		})
	}
}

func TestProvisioning_RKE2DigitalOceanCluster(t *testing.T) {
	for userName, bearerToken := range clients.BearerTokensList() {
		t.Logf("Setup for %s", userName)
		setupDigitalOcean := func() (*unstructured.Unstructured, string, *coreV1.Secret, *cloudcredentials.CloudCredential, error) {
			t.Log("Test set up")
			t.Log("Create Cloud Credential")
			client, err := clients.NewCoreV1Client(bearerToken)
			if err != nil {
				return nil, "", nil, nil, err
			}

			cloudCredentialName := digitalOceanCloudCredentialName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
			doCloudCred := cloudcredentials.NewCloudCredentialSecret(cloudCredentialName, "", "digitalocean", namespace)

			cloudCredential := cloudcredentials.NewCloudCredential(client)
			_, err = cloudCredential.CreateCloudCredential(doCloudCred)
			if err != nil {
				return nil, "", nil, nil, err
			}
			t.Log("Cloud Credential was created successfully")

			generatedPoolName := fmt.Sprintf("nc-%s-pool1", "digitalocean")

			machinePoolConfig := machinepool.NewMachinePoolConfig(generatedPoolName, machinepool.DOKind, namespace, machinepool.DOPoolType, "ubuntu-20-04-x64", "nyc3", "s-2vcpu-4gb")

			t.Logf("Creating DO machine pool config %s", generatedPoolName)
			podConfigClient, err := clients.NewPodConfigClient(clients.DOResourceConfig, bearerToken)
			require.NoError(t, err)

			machineConfigResult, err := machinepool.CreateMachineConfigPool(machinePoolConfig, podConfigClient)
			if err != nil {
				return nil, "", nil, nil, err
			}

			t.Logf("Successfully created DO machine pool %s", generatedPoolName)

			return machineConfigResult, cloudCredentialName, doCloudCred, cloudCredential, err
		}

		machineConfigResult, cloudCredentialName, doCloudCred, cloudCredential, setupErr := setupDigitalOcean()

		nodeRoles0 := [][]bool{
			{
				//control plane role
				true,
				//etcd role
				true,
				//worker role
				true,
			},
		}

		nodeRoles1 := [][]bool{
			{
				//control plane role
				true,
				//etcd role
				false,
				//worker role
				false,
			},
			{
				//control plane role
				false,
				//etcd role
				true,
				//worker role
				false,
			},
			{
				//control plane role
				false,
				//etcd role
				false,
				//worker role
				true,
			},
		}

		tests := []struct {
			name      string
			nodeRoles [][]bool
		}{
			{"1 Node all roles", nodeRoles0},
			{"3 nodes - 1 role per node", nodeRoles1},
		}

		var clusterNames []string
		var clusters []*cluster.Cluster
		for _, tt := range tests {
			name := tt.name + " " + userName
			t.Run(name, func(t *testing.T) {
				if setupErr != nil {
					t.Fatalf("Setup has failed with error: %v", setupErr)
				}
				//randomize name
				clusterName := baseDOClusterName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
				clusterNames = append(clusterNames, clusterName)

				provisioningClient, err := clients.NewProvisioningClient(bearerToken)
				require.NoError(t, err)

				machinePools := []apisV1.RKEMachinePool{}
				for index, roles := range tt.nodeRoles {
					machinePool := machinepool.MachinePoolSetup(roles[0], roles[1], roles[2], "pool"+strconv.Itoa(index), 1, machineConfigResult)
					machinePools = append(machinePools, machinePool)
				}

				clusterConfig := cluster.NewRKE2ClusterConfig(clusterName, namespace, cluster.CNI, cloudCredentialName, cluster.KubernetesVersion, machinePools)

				clusterObj := cluster.NewCluster(namespace, provisioningClient)
				clusters = append(clusters, clusterObj)

				t.Logf("Creating Cluster %s", clusterName)
				v1Cluster, err := clusterObj.CreateCluster(clusterConfig)
				require.NoError(t, err)

				assert.Equal(t, v1Cluster.Name, clusterName)

				t.Logf("Created Cluster %s", v1Cluster.ClusterName)

				t.Logf("Checking status of cluster %s", v1Cluster.ClusterName)
				//check cluster status
				ready, err := clusterObj.CheckClusterStatus(clusterName)
				assert.NoError(t, err)
				assert.True(t, ready)

				if environmentvariables.RancherCleanup() {
					err := cluster.ClusterCleanup(nil, clusters, clusterNames, nil)
					require.NoError(t, err)
				}
			})
		}

		// temporary fix to wait for rancher to delete Digital Ocean droplets before deleting the cloud credential.
		// this is to avoid the situation where deleting the cloud credential too soon would result in the Digital Ocean droplets not being deleted.
		time.Sleep(30 * time.Second)
		err := cloudCredential.DeleteCloudCredential(doCloudCred)
		require.NoError(t, err)
	}
}

func TestProvisioning_RKE2DigitalOceanClusterDynamicInput(t *testing.T) {
	if nodesAndRoles == "" {
		t.Skip()
	}

	for userName, bearerToken := range clients.BearerTokensList() {
		t.Run(userName, func(t *testing.T) {
			t.Log("Test set up")
			t.Log("Create Cloud Credential")
			client, err := clients.NewCoreV1Client(bearerToken)
			require.NoError(t, err)

			cloudCredentialName := digitalOceanCloudCredentialName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
			doCloudCred := cloudcredentials.NewCloudCredentialSecret(cloudCredentialName, "", "digitalocean", namespace)

			cloudCredential := cloudcredentials.NewCloudCredential(client)
			_, err = cloudCredential.CreateCloudCredential(doCloudCred)
			require.NoError(t, err)
			t.Log("Cloud Credential was created successfully")

			generatedPoolName := fmt.Sprintf("nc-%s-pool1", "digitalocean")

			machinePoolConfig := machinepool.NewMachinePoolConfig(generatedPoolName, machinepool.DOKind, namespace, machinepool.DOPoolType, "ubuntu-20-04-x64", "nyc3", "s-2vcpu-4gb")

			t.Logf("Creating DO machine pool config %s", generatedPoolName)
			podConfigClient, err := clients.NewPodConfigClient(clients.DOResourceConfig, bearerToken)
			require.NoError(t, err)

			machineConfigResult, err := machinepool.CreateMachineConfigPool(machinePoolConfig, podConfigClient)
			require.NoError(t, err)

			t.Logf("Successfully created DO machine pool %s", generatedPoolName)

			nodeRolesBoolSliceMap := []map[string]bool{}

			rolesSlice := strings.Split(nodesAndRoles, "|")
			for _, roles := range rolesSlice {
				nodeRoles := strings.Split(roles, ",")
				nodeRoleBoolMap := map[string]bool{}
				for _, nodeRole := range nodeRoles {
					nodeRoleBoolMap[nodeRole] = true

				}
				nodeRolesBoolSliceMap = append(nodeRolesBoolSliceMap, nodeRoleBoolMap)
			}

			var clusterNames []string
			var clusters []*cluster.Cluster

			//randomize name
			clusterName := baseDOClusterName + namegenerator.RandStringLowerBytes(defaultRandStringLength)
			clusterNames = append(clusterNames, clusterName)

			provisioningClient, err := clients.NewProvisioningClient(bearerToken)
			require.NoError(t, err)

			machinePools := []apisV1.RKEMachinePool{}
			for index, roles := range nodeRolesBoolSliceMap {
				machinePool := machinepool.MachinePoolSetup(roles["controlplane"], roles["etcd"], roles["worker"], "pool"+strconv.Itoa(index), 1, machineConfigResult)
				machinePools = append(machinePools, machinePool)
			}

			clusterConfig := cluster.NewRKE2ClusterConfig(clusterName, namespace, cluster.CNI, cloudCredentialName, cluster.KubernetesVersion, machinePools)

			clusterObj := cluster.NewCluster(namespace, provisioningClient)
			clusters = append(clusters, clusterObj)

			t.Logf("Creating Cluster %s", clusterName)
			v1Cluster, err := clusterObj.CreateCluster(clusterConfig)
			require.NoError(t, err)

			assert.Equal(t, v1Cluster.Name, clusterName)

			t.Logf("Created Cluster %s", v1Cluster.ClusterName)

			t.Logf("Checking status of cluster %s", v1Cluster.ClusterName)
			//check cluster status
			ready, err := clusterObj.CheckClusterStatus(clusterName)
			assert.NoError(t, err)
			assert.True(t, ready)

			if environmentvariables.RancherCleanup() {
				err := cluster.ClusterCleanup(nil, clusters, clusterNames, nil)
				require.NoError(t, err)

				// temporary fix to wait for rancher to delete Digital Ocean droplets before deleting the cloud credential.
				// this is to avoid the situation where deleting the cloud credential too soon would result in the Digital Ocean droplets not being deleted.
				time.Sleep(30 * time.Second)
				err = cloudCredential.DeleteCloudCredential(doCloudCred)
				require.NoError(t, err)
			}
		})
	}
}

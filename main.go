package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computefleet/armcomputefleet/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computeschedule/armcomputeschedule"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/google/uuid"
)

const (
	subscriptionId    = "31352ba3-4576-4d93-9214-7c2d18b24067"
	resourceGroupName = "DEMO-GO-SDK-RG"
	vnetName          = "DEMO-VN"
	location          = "uksouth"
)

var (
	conn azcore.TokenCredential
	ctx  context.Context
	err  error
)

var (
	resourcesClientFactory       *armresources.ClientFactory
	networkClientFactory         *armnetwork.ClientFactory
	computeFleetClientFactory    *armcomputefleet.ClientFactory
	computeScheduleClientFactory *armcomputeschedule.ClientFactory
	computeClientFactory         *armcompute.ClientFactory
)

var (
	resourceGroupClient    *armresources.ResourceGroupsClient
	providerClient         *armresources.ProvidersClient
	virtualNetworksClient  *armnetwork.VirtualNetworksClient
	fleetsClient           *armcomputefleet.FleetsClient
	scheduledActionsClient *armcomputeschedule.ScheduledActionsClient
	virtualMachinesClient  *armcompute.VirtualMachinesClient
)

func main() {
	// Set up the environment, create the clients, the resource group and virtual network
	setup()

	// Create 1K VMs of Regular priority using BulkActions
	createBulkActions(
		/*bulkActionsName*/ "BA-1K-VMs",
		/*capacityType*/ armcomputefleet.CapacityTypeVM,
		/*spotCapacity*/ 0,
		/*regularCapacity*/ 1000,
		/*vmSizesProfile*/ []*armcomputefleet.VMSizeProfile{
			{Name: to.Ptr("Standard_F1s")},
			{Name: to.Ptr("Standard_DS1_v2")},
			{Name: to.Ptr("Standard_D2ads_v5")},
			{Name: to.Ptr("Standard_D8as_v5")},
		},
		/*vmAttributes*/ nil)

	// Create 1K VCPUs of Regular priority using BulkActions
	createBulkActions(
		/*bulkActionsName*/ "BA-1K-VCPUs",
		/*capacityType*/ armcomputefleet.CapacityTypeVCPU,
		/*spotCapacity*/ 0,
		/*regularCapacity*/ 1000,
		/*vmSizesProfile*/ []*armcomputefleet.VMSizeProfile{
			{Name: to.Ptr("Standard_F2s")},
			{Name: to.Ptr("Standard_DS2_v2")},
			{Name: to.Ptr("Standard_E2s_v3")},
			{Name: to.Ptr("Standard_D2as_v4")},
		},
		/*vmAttributes*/ nil)

	// Create 2K VCPUs of Spot priority with the specified VM Attributes using BulkActions
	createBulkActions(
		/*bulkActionsName*/ "BA-ATTRIBUTES-2K-VCPUs",
		/*capacityType*/ armcomputefleet.CapacityTypeVCPU,
		/*spotCapacity*/ 2000,
		/*regularCapacity*/ 0,
		/*vmSizesProfile*/ nil,
		/*vmAttributes*/ &armcomputefleet.VMAttributes{
			VCPUCount: &armcomputefleet.VMAttributeMinMaxInteger{
				Min: to.Ptr[int32](1),
				Max: to.Ptr[int32](64),
			},
			MemoryInGiB: &armcomputefleet.VMAttributeMinMaxDouble{
				Min: to.Ptr(8.0),
				Max: to.Ptr(256.0),
			},
			MemoryInGiBPerVCpu: &armcomputefleet.VMAttributeMinMaxDouble{
				Min: to.Ptr(8.0),
				Max: to.Ptr(8.0),
			},
		})

	// Register the subscription with the Microsoft.ComputeSchedule resource provider
	// This is required in order to delete VMs using the ScheduledActions BulkDelete feature
	providerClient.Register(ctx, "Microsoft.ComputeSchedule", nil)

	// Get the list of VMs in the first BulkActions
	ba_1k_vms_list := listVMsInBulkAction("BA-1K-VMs")

	// Delete all VMs created by the first BulkActions using the ScheduledActions BulkDelete feature with forceDelete
	bulkDeleteVMsInBatch(
		ba_1k_vms_list,
		/*forceDelete*/ true,
	)

	// Get the list of VMs in the second BulkActions
	ba_1k_vcpus_list := listVMsInBulkAction("BA-1K-VCPUs")

	// Delete half of the VMs created by the second BulkActions using the ScheduledActions BulkDelete feature with forceDelete
	bulkDeleteVMsInBatch(
		ba_1k_vcpus_list[:len(ba_1k_vcpus_list)/2],
		/*forceDelete*/ true,
	)

	// Get the list of all remaining VMs in the resource group (half of the second BulkActions and all of the third BulkActions)
	remaining_vms_list := listVMsInRG()

	// Delete all remaining VMs in the resource group using the ScheduledActions BulkDelete feature with forceDelete
	bulkDeleteVMsInBatch(
		remaining_vms_list,
		/*forceDelete*/ true,
	)

	// Delete RG to cleanup all BulkActions
	deleteResourceGroup()
}

func setup() {
	conn = authenticate()
	ctx = context.Background()

	resourcesClientFactory, err = armresources.NewClientFactory(subscriptionId, conn, nil)
	logIfError(err)

	resourceGroupClient = resourcesClientFactory.NewResourceGroupsClient()

	networkClientFactory, err = armnetwork.NewClientFactory(subscriptionId, conn, nil)
	logIfError(err)

	virtualNetworksClient = networkClientFactory.NewVirtualNetworksClient()

	computeFleetClientFactory, err = armcomputefleet.NewClientFactory(subscriptionId, conn, nil)
	logIfError(err)

	fleetsClient = computeFleetClientFactory.NewFleetsClient()

	providerClient = resourcesClientFactory.NewProvidersClient()

	computeScheduleClientFactory, err = armcomputeschedule.NewClientFactory(subscriptionId, conn, nil)
	scheduledActionsClient = computeScheduleClientFactory.NewScheduledActionsClient()

	computeClientFactory, err = armcompute.NewClientFactory(subscriptionId, conn, nil)
	logIfError(err)

	virtualMachinesClient = computeClientFactory.NewVirtualMachinesClient()

	createResourceGroup()
	createVirtualNetwork()
}

func createResourceGroup() {
	log.Printf("Creating resource group %s...", resourceGroupName)

	parameters := armresources.ResourceGroup{
		Location: to.Ptr(location),
	}

	resourceGroupClient.CreateOrUpdate(ctx, resourceGroupName, parameters, nil)

	log.Printf("Created resource group: %s", resourceGroupName)
}

func deleteResourceGroup() {
	log.Printf("Deleting resource group %s...", resourceGroupName)

	resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)

	log.Printf("Deleted resource group: %s", resourceGroupName)
}

func createVirtualNetwork() {
	log.Printf("Creating virtual network %s...", vnetName)

	parameters := armnetwork.VirtualNetwork{
		Location: to.Ptr(location),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					to.Ptr("10.1.0.0/16"),
				},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: to.Ptr("default"),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: to.Ptr("10.1.0.0/18"),
					},
				},
			},
		},
	}

	virtualNetworksClient.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, parameters, nil)

	log.Printf("Created virtual network %s", vnetName)
}

func createBulkActions(
	bulkActionsName string,
	capacityType armcomputefleet.CapacityType,
	spotCapacity int32,
	regularCapacity int32,
	vmSizesProfile []*armcomputefleet.VMSizeProfile,
	vmAttributes *armcomputefleet.VMAttributes) {
	log.Printf("Creating BulkActions %s...", bulkActionsName)

	parameters := armcomputefleet.Fleet{
		Location: to.Ptr(location),
		Properties: &armcomputefleet.FleetProperties{
			Mode:         to.Ptr(armcomputefleet.FleetModeInstance),
			CapacityType: to.Ptr(capacityType),
			SpotPriorityProfile: &armcomputefleet.SpotPriorityProfile{
				Capacity: to.Ptr(spotCapacity),
			},
			RegularPriorityProfile: &armcomputefleet.RegularPriorityProfile{
				Capacity: to.Ptr(regularCapacity),
			},
			VMSizesProfile: vmSizesProfile,
			VMAttributes:   vmAttributes,
			ComputeProfile: &armcomputefleet.ComputeProfile{
				BaseVirtualMachineProfile: &armcomputefleet.BaseVirtualMachineProfile{
					StorageProfile: &armcomputefleet.VirtualMachineScaleSetStorageProfile{
						ImageReference: &armcomputefleet.ImageReference{
							Publisher: to.Ptr("Canonical"),
							Offer:     to.Ptr("ubuntu-24_04-lts"),
							SKU:       to.Ptr("server-gen1"),
							Version:   to.Ptr("latest"),
						},
						OSDisk: &armcomputefleet.VirtualMachineScaleSetOSDisk{
							OSType:       to.Ptr(armcomputefleet.OperatingSystemTypesLinux),
							CreateOption: to.Ptr(armcomputefleet.DiskCreateOptionTypesFromImage),
							DeleteOption: to.Ptr(armcomputefleet.DiskDeleteOptionTypesDelete),
							Caching:      to.Ptr(armcomputefleet.CachingTypesReadWrite),
							ManagedDisk: &armcomputefleet.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: to.Ptr(armcomputefleet.StorageAccountTypesStandardLRS),
							},
						},
					},
					OSProfile: &armcomputefleet.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.Ptr("sample-compute"),
						AdminUsername:      to.Ptr("sample-user"),
						AdminPassword:      to.Ptr("***********"), // use a valid password here
					},
					NetworkProfile: &armcomputefleet.VirtualMachineScaleSetNetworkProfile{
						NetworkAPIVersion: to.Ptr(armcomputefleet.NetworkAPIVersionV20201101),
						NetworkInterfaceConfigurations: []*armcomputefleet.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.Ptr("nic"),
								Properties: &armcomputefleet.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary:            to.Ptr(true),
									EnableIPForwarding: to.Ptr(true),
									IPConfigurations: []*armcomputefleet.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.Ptr("ip"),
											Properties: &armcomputefleet.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &armcomputefleet.APIEntityReference{
													ID: to.Ptr("/subscriptions/" + subscriptionId + "/resourceGroups/" + resourceGroupName + "/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/default"),
												},
												Primary: to.Ptr(true),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	poller, err := fleetsClient.BeginCreateOrUpdate(ctx, resourceGroupName, bulkActionsName, parameters, nil)
	logIfError(err)

	res, err := poller.PollUntilDone(ctx, nil)
	logIfError(err)

	// You could use response here. We use blank identifier for just demo purposes.
	_ = res

	log.Printf("Created BulkActions %s", bulkActionsName)
}

func listVMsInBulkAction(bulkActionsName string) []*string {
	var allVMs []*string
	pager := fleetsClient.NewListVirtualMachinesPager(
		resourceGroupName,
		bulkActionsName,
		/*options*/ nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		logIfError(err)

		for _, v := range page.Value {
			allVMs = append(allVMs, v.ID)
		}
	}

	log.Printf("Total VMs found in BulkActions %s: %d", bulkActionsName, len(allVMs))

	return allVMs
}

func listVMsInRG() []*string {
	var allVMs []*string
	pager := virtualMachinesClient.NewListPager(
		resourceGroupName,
		/*options*/ nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		logIfError(err)

		for _, v := range page.Value {
			allVMs = append(allVMs, v.ID)
		}
	}

	log.Printf("Total VMs found in RG %s: %d", resourceGroupName, len(allVMs))

	return allVMs
}

func bulkDeleteVMsInBatch(vmIDs []*string, forceDelete bool) {
	var wg sync.WaitGroup

	for _, batch := range batch(vmIDs, 100) {
		wg.Add(1)
		go func(batch []*string) {
			defer wg.Done()
			bulkDeleteVMs(batch, forceDelete)
		}(batch)
	}

	wg.Wait()
}

func bulkDeleteVMs(vmIDs []*string, forceDelete bool) {
	log.Printf("Starting bulk deletion of %d VMs", len(vmIDs))

	// Start deletion
	deleteResp, err := scheduledActionsClient.VirtualMachinesExecuteDelete(ctx, location, armcomputeschedule.ExecuteDeleteRequest{
		ExecutionParameters: &armcomputeschedule.ExecutionParameters{},
		Resources:           &armcomputeschedule.Resources{IDs: vmIDs},
		Correlationid:       to.Ptr(uuid.New().String()),
		ForceDeletion:       to.Ptr(forceDelete),
	}, nil)
	logIfError(err)

	// Poll until completion
	pollingCompleted := false
	for !pollingCompleted {
		statusResp, err := scheduledActionsClient.VirtualMachinesGetOperationStatus(ctx, location, getOpsRequest(deleteResp), nil)
		logIfError(err)

		pollingCompleted = isPollingComplete(statusResp)
		time.Sleep(30 * time.Second)
	}

	log.Printf("Completed bulk deletion of %d VMs", len(vmIDs))
}

func getOpsRequest(
	deleteResponse armcomputeschedule.ScheduledActionsClientVirtualMachinesExecuteDeleteResponse) armcomputeschedule.GetOperationStatusRequest {
	var operationIds []*string

	for _, result := range deleteResponse.Results {
		operation := result.Operation
		if operation != nil && operation.OperationID != nil {
			operationIds = append(operationIds, operation.OperationID)
		}
	}

	return armcomputeschedule.GetOperationStatusRequest{
		OperationIDs:  operationIds,
		Correlationid: to.Ptr(uuid.New().String()),
	}
}

func isPollingComplete(getOpsResponse armcomputeschedule.ScheduledActionsClientVirtualMachinesGetOperationStatusResponse) bool {
	for _, result := range getOpsResponse.Results {
		operation := result.Operation
		if operation == nil || operation.State == nil || *operation.State != armcomputeschedule.OperationStateSucceeded {
			return false
		}
	}
	return true
}

func authenticate() azcore.TokenCredential {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	logIfError(err)
	return cred
}

func logIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func batch[T any](input []T, batchSize int) [][]T {
	var batches [][]T

	for i := 0; i < len(input); i += batchSize {
		end := i + batchSize
		if end > len(input) {
			end = len(input)
		}
		batches = append(batches, input[i:end])
	}

	return batches
}

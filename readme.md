# BulkActions + ScheduledActions Demo ([Go SDK](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computefleet/armcomputefleet/v2))

This repository contains an end-to-end demo that shows how to create, scale, and delete VMs using the **[Azure Go SDK](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computefleet/armcomputefleet/v2)**.  
It demonstrates:  
- **BulkActions API** for VM creation and scaling  
- **ScheduledActions Bulk Delete API** for VM deletion  

---

## Prerequisites

- [Go 1.21+](https://go.dev/dl/) installed locally  
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) (for login)  

---

## Setup

1. Clone this repository:
   ```bash
   git clone https://github.com/tissabre/bulkactions-go-sdk-demo.git
   cd bulkactions-go-sdk-demo
   ```

2. Authenticate with Azure CLI:
   ```bash
   az login
   ```

3. Build and run the program
   ```bash
   go run main.go
   ```

## More Examples

Take a look at the [SDK source repo](https://github.com/Azure/azure-sdk-for-go/tree/sdk/resourcemanager/computefleet/armcomputefleet/v2.0.0-beta.1/sdk/resourcemanager/computefleet/armcomputefleet) for more [code samples](https://github.com/Azure/azure-sdk-for-go/blob/sdk/resourcemanager/computefleet/armcomputefleet/v2.0.0-beta.1/sdk/resourcemanager/computefleet/armcomputefleet/fleets_client_example_test.go)!
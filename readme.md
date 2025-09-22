# BulkActions + ScheduledActions Demo (Go SDK)

This repository contains an end-to-end demo that shows how to create, scale, and delete VMs using the **Azure Go SDK**.  
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
// Objects read by this setup program (not created, IDs stored as env vars):
//
// DNS Hosts (up to 2):
//   - UDDI_DNS_HOST_ID_1, UDDI_DNS_HOST_ID_2
//
// DHCP Hosts (up to 2):
//   - UDDI_DHCP_HOST_ID_1, UDDI_DHCP_HOST_ID_2
//
// Objects created by this setup program (IDs stored as env vars):
//
// DHCP Option Groups:
//   - tf_test_option_group_1 (UDDI_OPTION_GROUP_1_ID)
//   - tf_test_option_group_2 (UDDI_OPTION_GROUP_2_ID)

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	uddiclient "github.com/infobloxopen/universal-ddi-go-client/client"
	"github.com/infobloxopen/universal-ddi-go-client/ipam"
	uddioption "github.com/infobloxopen/universal-ddi-go-client/option"
)

var pipelineEnvFile *os.File

func writePipelineEnvVar(key, value string) error {
	if pipelineEnvFile == nil {
		return fmt.Errorf("pipeline.env file is not initialized")
	}
	if _, err := fmt.Fprintf(pipelineEnvFile, "%s=%s\n", key, value); err != nil {
		return fmt.Errorf("write env var %s to pipeline.env: %w", key, err)
	}
	return nil
}

// StoreDNSHostIDs lists DNS Host objects and stores the IDs of the first two
// into pipeline.env as UDDI_DNS_HOST_ID_1 and UDDI_DNS_HOST_ID_2.
func StoreDNSHostIDs(ctx context.Context, client *uddiclient.APIClient) error {
	resp, _, err := client.DNSConfigurationAPI.HostAPI.List(ctx).Execute()
	if err != nil {
		return fmt.Errorf("store DNS host IDs: list DNS hosts: %w", err)
	}

	if resp == nil || resp.Results == nil || len(resp.Results) == 0 {
		fmt.Println("No DNS hosts found, skipping DNS host ID storage")
		return nil
	}

	envVars := []string{"UDDI_DNS_HOST_ID_1", "UDDI_DNS_HOST_ID_2"}
	stored := 0

	for i, host := range resp.Results {
		if i >= 2 {
			break
		}
		if host.Id == nil || *host.Id == "" {
			fmt.Printf("DNS host at index %d has no ID, skipping\n", i)
			continue
		}
		if err := writePipelineEnvVar(envVars[stored], *host.Id); err != nil {
			return fmt.Errorf("store DNS host IDs: write %s: %w", envVars[stored], err)
		}
		fmt.Printf("Stored DNS host ID %q as %s\n", *host.Id, envVars[stored])
		stored++
	}

	if stored == 0 {
		fmt.Println("No DNS host IDs could be stored (all hosts missing ID)")
	}

	return nil
}

// StoreDHCPHostIDs lists DHCP Host objects and stores the IDs of the first two
// into pipeline.env as UDDI_DHCP_HOST_ID_1 and UDDI_DHCP_HOST_ID_2.
func StoreDHCPHostIDs(ctx context.Context, client *uddiclient.APIClient) error {
	resp, _, err := client.IPAddressManagementAPI.DhcpHostAPI.List(ctx).Execute()
	if err != nil {
		return fmt.Errorf("store DHCP host IDs: list DHCP hosts: %w", err)
	}

	if resp == nil || resp.Results == nil || len(resp.Results) == 0 {
		fmt.Println("No DHCP hosts found, skipping DHCP host ID storage")
		return nil
	}

	envVars := []string{"UDDI_DHCP_HOST_ID_1", "UDDI_DHCP_HOST_ID_2"}
	stored := 0

	for i, host := range resp.Results {
		if i >= 2 {
			break
		}
		if host.Id == nil || *host.Id == "" {
			fmt.Printf("DHCP host at index %d has no ID, skipping\n", i)
			continue
		}
		if err := writePipelineEnvVar(envVars[stored], *host.Id); err != nil {
			return fmt.Errorf("store DHCP host IDs: write %s: %w", envVars[stored], err)
		}
		fmt.Printf("Stored DHCP host ID %q as %s\n", *host.Id, envVars[stored])
		stored++
	}

	if stored == 0 {
		fmt.Println("No DHCP host IDs could be stored (all hosts missing ID)")
	}

	return nil
}

// CreateOptionGroups creates two DHCP option groups and stores their IDs into
// pipeline.env as UDDI_OPTION_GROUP_1_ID and UDDI_OPTION_GROUP_2_ID.
// If a group already exists, its existing ID is stored instead.
func CreateOptionGroups(ctx context.Context, client *uddiclient.APIClient) error {
	optionGroups := []struct {
		name  string
		idVar string
	}{
		{name: "tf_test_option_group_1", idVar: "UDDI_OPTION_GROUP_1_ID"},
		{name: "tf_test_option_group_2", idVar: "UDDI_OPTION_GROUP_2_ID"},
	}

	for _, og := range optionGroups {
		body := ipam.OptionGroup{
			Name: og.name,
		}

		resp, _, err := client.IPAddressManagementAPI.OptionGroupAPI.Create(ctx).Body(body).Execute()
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "conflict") {
				// Fetch the existing group's ID
				listResp, _, listErr := client.IPAddressManagementAPI.OptionGroupAPI.List(ctx).Execute()
				if listErr != nil {
					return fmt.Errorf("create option groups: list existing groups to find %q: %w", og.name, listErr)
				}

				var existingID string
				if listResp != nil {
					for _, existing := range listResp.Results {
						if existing.Name == og.name && existing.Id != nil {
							existingID = *existing.Id
							break
						}
					}
				}

				if existingID == "" {
					return fmt.Errorf("create option groups: option group %q already exists but ID could not be resolved", og.name)
				}

				if err := writePipelineEnvVar(og.idVar, existingID); err != nil {
					return fmt.Errorf("create option groups: write %s for existing group: %w", og.idVar, err)
				}

				fmt.Printf("Option group %q already exists, using existing ID %q (env: %s)\n", og.name, existingID, og.idVar)
				continue
			}
			return fmt.Errorf("create option groups: create %q: %w", og.name, err)
		}

		if resp == nil || resp.Result == nil || resp.Result.Id == nil {
			return fmt.Errorf("create option groups: create response for %q missing ID", og.name)
		}

		createdID := *resp.Result.Id
		if err := writePipelineEnvVar(og.idVar, createdID); err != nil {
			return fmt.Errorf("create option groups: write %s: %w", og.idVar, err)
		}

		fmt.Printf("Option group %q created successfully (ID: %q, env: %s)\n", og.name, createdID, og.idVar)
	}

	return nil
}

func main() {
	cspURL := strings.TrimSpace(os.Getenv("INFOBLOX_PORTAL_URL"))
	apiKey := strings.TrimSpace(os.Getenv("INFOBLOX_PORTAL_KEY"))

	if cspURL == "" || apiKey == "" {
		fmt.Println("Missing required UDDI configuration.")
		fmt.Println("Supported env vars: INFOBLOX_PORTAL_URL, INFOBLOX_PORTAL_KEY")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	pipelineEnvPath := filepath.Join(cwd, "pipeline.env")
	f, err := os.OpenFile(pipelineEnvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Error opening pipeline.env: %v\n", err)
		return
	}
	pipelineEnvFile = f
	defer func() {
		_ = pipelineEnvFile.Close()
		pipelineEnvFile = nil
	}()

	client := uddiclient.NewAPIClient(
		uddioption.WithClientName("terraform-integration-test-setup"),
		uddioption.WithCSPUrl(cspURL),
		uddioption.WithAPIKey(apiKey),
	)

	ctx := context.Background()

	if err := StoreDNSHostIDs(ctx, client); err != nil {
		fmt.Printf("Error storing DNS host IDs: %v\n", err)
		return
	}
	fmt.Println("DNS host IDs stored successfully")

	if err := StoreDHCPHostIDs(ctx, client); err != nil {
		fmt.Printf("Error storing DHCP host IDs: %v\n", err)
		return
	}
	fmt.Println("DHCP host IDs stored successfully")

	if err := CreateOptionGroups(ctx, client); err != nil {
		fmt.Printf("Error creating option groups: %v\n", err)
		return
	}
	fmt.Println("Option groups created successfully")
}

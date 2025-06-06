package kubevirt

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// NetworkHandler handles VM networking operations and address extraction
type NetworkHandler struct{}

// NewNetworkHandler creates a new NetworkHandler instance
func NewNetworkHandler() *NetworkHandler {
	return &NetworkHandler{}
}

// ExtractAddresses extracts IP addresses from a VMI and returns them in Nova API format
func (n *NetworkHandler) ExtractAddresses(vmi *unstructured.Unstructured) map[string][]models.Address {
	addresses := make(map[string][]models.Address)
	
	if vmi == nil {
		return addresses
	}
	
	interfaces, found, _ := unstructured.NestedSlice(vmi.Object, "status", "interfaces")
	if !found {
		return addresses
	}
	
	defaultAddresses := n.parseInterfaces(interfaces)
	if len(defaultAddresses) > 0 {
		addresses[DefaultNetworkName] = defaultAddresses
	}
	
	return addresses
}

// parseInterfaces parses the VMI interfaces and extracts IP addresses
func (n *NetworkHandler) parseInterfaces(interfaces []interface{}) []models.Address {
	var addresses []models.Address
	
	for _, iface := range interfaces {
		ifaceMap, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}
		
		// Extract IP address from interface
		if ip, found := ifaceMap["ipAddress"].(string); found && ip != "" {
			addresses = append(addresses, models.Address{
				Version: 4,
				Addr:    ip,
				Type:    "fixed",
			})
		}
		
		// Also check for IPv6 addresses
		if ipv6, found := ifaceMap["ipv6Address"].(string); found && ipv6 != "" {
			addresses = append(addresses, models.Address{
				Version: 6,
				Addr:    ipv6,
				Type:    "fixed",
			})
		}
		
		// Extract additional IPs if present
		if ips, found := ifaceMap["ips"].([]interface{}); found {
			for _, ipInterface := range ips {
				if ip, ok := ipInterface.(string); ok && ip != "" {
					// Determine IP version
					version := 4
					if n.isIPv6(ip) {
						version = 6
					}
					
					addresses = append(addresses, models.Address{
						Version: version,
						Addr:    ip,
						Type:    "fixed",
					})
				}
			}
		}
	}
	
	return addresses
}

// isIPv6 checks if an IP address is IPv6
func (n *NetworkHandler) isIPv6(ip string) bool {
	return len(ip) > 15 && (ip[0] == ':' || ip[1] == ':' || ip[2] == ':' || 
		ip[3] == ':' || ip[4] == ':' || ip[5] == ':' || 
		containsColon(ip))
}

// containsColon checks if string contains colon (simple IPv6 detection)
func containsColon(s string) bool {
	for _, c := range s {
		if c == ':' {
			return true
		}
	}
	return false
}

// BuildDefaultNetworkSpec builds the default network specification for a VM
func (n *NetworkHandler) BuildDefaultNetworkSpec() map[string]interface{} {
	return map[string]interface{}{
		"name": DefaultNetworkName,
		"pod":  map[string]interface{}{},
	}
}

// BuildDefaultInterfaceSpec builds the default interface specification for a VM
func (n *NetworkHandler) BuildDefaultInterfaceSpec() map[string]interface{} {
	return map[string]interface{}{
		"name":       DefaultNetworkName,
		"masquerade": map[string]interface{}{},
	}
}

// GetNetworkInfo returns network information for a VM
func (n *NetworkHandler) GetNetworkInfo(vmi *unstructured.Unstructured) (*NetworkInfo, error) {
	if vmi == nil {
		return &NetworkInfo{
			Networks:   []string{},
			Addresses:  map[string][]models.Address{},
			Status:     "disconnected",
		}, nil
	}
	
	addresses := n.ExtractAddresses(vmi)
	networks := n.extractNetworkNames(vmi)
	status := n.getNetworkStatus(vmi)
	
	return &NetworkInfo{
		Networks:   networks,
		Addresses:  addresses,
		Status:     status,
	}, nil
}

// NetworkInfo represents network information for a VM
type NetworkInfo struct {
	Networks  []string                        `json:"networks"`
	Addresses map[string][]models.Address     `json:"addresses"`
	Status    string                          `json:"status"`
}

// extractNetworkNames extracts network names from VMI spec
func (n *NetworkHandler) extractNetworkNames(vmi *unstructured.Unstructured) []string {
	var networks []string
	
	networkList, found, _ := unstructured.NestedSlice(vmi.Object, "spec", "networks")
	if !found {
		return networks
	}
	
	for _, network := range networkList {
		networkMap, ok := network.(map[string]interface{})
		if !ok {
			continue
		}
		
		if name, found := networkMap["name"].(string); found {
			networks = append(networks, name)
		}
	}
	
	return networks
}

// getNetworkStatus determines the network status of a VMI
func (n *NetworkHandler) getNetworkStatus(vmi *unstructured.Unstructured) string {
	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return "unknown"
	}
	
	switch phase {
	case VMStateRunning:
		// Check if interfaces are ready
		interfaces, found, _ := unstructured.NestedSlice(vmi.Object, "status", "interfaces")
		if found && len(interfaces) > 0 {
			return "connected"
		}
		return "connecting"
	case VMStatePending, VMStateScheduling, VMStateScheduled:
		return "connecting"
	case VMStateFailed:
		return "error"
	default:
		return "disconnected"
	}
}

// ValidateNetworkConfiguration validates network configuration for VM creation
func (n *NetworkHandler) ValidateNetworkConfiguration(networks []models.Network) error {
	if len(networks) == 0 {
		// No networks specified, will use default
		return nil
	}
	
	for i, network := range networks {
		if err := n.validateSingleNetwork(network, i); err != nil {
			return err
		}
	}
	
	return nil
}

// validateSingleNetwork validates a single network configuration
func (n *NetworkHandler) validateSingleNetwork(network models.Network, index int) error {
	if network.UUID == "" && network.Port == "" {
		return fmt.Errorf("network at index %d must specify either UUID or port", index)
	}
	
	if network.UUID != "" && network.Port != "" {
		return fmt.Errorf("network at index %d cannot specify both UUID and port", index)
	}
	
	if network.FixedIP != "" {
		// Basic IP format validation could be added here
		// For now, we just check it's not empty if specified
	}
	
	return nil
}

// BuildNetworksForVM builds network specifications for VM creation
func (n *NetworkHandler) BuildNetworksForVM(requestNetworks []models.Network) []map[string]interface{} {
	networks := make([]map[string]interface{}, 0)
	
	// Always add default network first
	networks = append(networks, n.BuildDefaultNetworkSpec())
	
	// Add additional networks if specified
	for _, network := range requestNetworks {
		if network.UUID != "" {
			// Handle Multus network attachment
			networks = append(networks, map[string]interface{}{
				"name": fmt.Sprintf("net-%s", network.UUID),
				"multus": map[string]interface{}{
					"networkName": network.UUID,
				},
			})
		}
	}
	
	return networks
}

// BuildInterfacesForVM builds interface specifications for VM creation
func (n *NetworkHandler) BuildInterfacesForVM(requestNetworks []models.Network) []map[string]interface{} {
	interfaces := make([]map[string]interface{}, 0)
	
	// Always add default interface first
	interfaces = append(interfaces, n.BuildDefaultInterfaceSpec())
	
	// Add additional interfaces if specified
	for _, network := range requestNetworks {
		if network.UUID != "" {
			interfaces = append(interfaces, map[string]interface{}{
				"name":   fmt.Sprintf("net-%s", network.UUID),
				"bridge": map[string]interface{}{},
			})
		}
	}
	
	return interfaces
}
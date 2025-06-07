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

// ExtractInternalAddresses extracts IP addresses from a VMI and returns them in internal format
func (n *NetworkHandler) ExtractInternalAddresses(vmi *unstructured.Unstructured) map[string][]models.InternalAddress {
	addresses := make(map[string][]models.InternalAddress)
	
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
func (n *NetworkHandler) parseInterfaces(interfaces []interface{}) []models.InternalAddress {
	var addresses []models.InternalAddress
	
	for _, iface := range interfaces {
		ifaceMap, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}
		
		// Extract IP address from interface
		if ip, found := ifaceMap["ipAddress"].(string); found && ip != "" {
			addresses = append(addresses, models.InternalAddress{
				Version: 4,
				IP:      ip,
				Type:    models.InternalAddressTypeInternal,
			})
		}
		
		// Also check for IPv6 addresses
		if ipv6, found := ifaceMap["ipv6Address"].(string); found && ipv6 != "" {
			addresses = append(addresses, models.InternalAddress{
				Version: 6,
				IP:      ipv6,
				Type:    models.InternalAddressTypeInternal,
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
					
					addresses = append(addresses, models.InternalAddress{
						Version: version,
						IP:      ip,
						Type:    models.InternalAddressTypeInternal,
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
			Addresses:  map[string][]models.InternalAddress{},
			Status:     "disconnected",
		}, nil
	}
	
	addresses := n.ExtractInternalAddresses(vmi)
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
	Networks  []string                            `json:"networks"`
	Addresses map[string][]models.InternalAddress `json:"addresses"`
	Status    string                              `json:"status"`
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
func (n *NetworkHandler) ValidateNetworkConfiguration(networks []models.InternalNetwork) error {
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
func (n *NetworkHandler) validateSingleNetwork(network models.InternalNetwork, index int) error {
	if network.Name == "" {
		return fmt.Errorf("network at index %d must specify a name", index)
	}
	
	if network.Type != "" {
		validTypes := []string{models.InternalNetworkTypePod, models.InternalNetworkTypeBridge, models.InternalNetworkTypeSRIOV}
		valid := false
		for _, validType := range validTypes {
			if network.Type == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("network at index %d has invalid type: %s", index, network.Type)
		}
	}
	
	return nil
}

// BuildNetworksForVM builds network specifications for VM creation
func (n *NetworkHandler) BuildNetworksForVM(requestNetworks []models.InternalNetwork) []map[string]interface{} {
	networks := make([]map[string]interface{}, 0)
	
	// Always add default network first
	networks = append(networks, n.BuildDefaultNetworkSpec())
	
	// Add additional networks if specified
	for _, network := range requestNetworks {
		if network.Name != "" && network.Name != DefaultNetworkName {
			// Handle additional network attachment
			netSpec := map[string]interface{}{
				"name": network.Name,
			}
			
			// Add network type specific configuration
			switch network.Type {
			case models.InternalNetworkTypeBridge:
				netSpec["bridge"] = map[string]interface{}{}
			case models.InternalNetworkTypeSRIOV:
				netSpec["sriov"] = map[string]interface{}{}
			default:
				netSpec["pod"] = map[string]interface{}{}
			}
			
			networks = append(networks, netSpec)
		}
	}
	
	return networks
}

// BuildInterfacesForVM builds interface specifications for VM creation
func (n *NetworkHandler) BuildInterfacesForVM(requestNetworks []models.InternalNetwork) []map[string]interface{} {
	interfaces := make([]map[string]interface{}, 0)
	
	// Always add default interface first
	interfaces = append(interfaces, n.BuildDefaultInterfaceSpec())
	
	// Add additional interfaces if specified
	for _, network := range requestNetworks {
		if network.Name != "" && network.Name != DefaultNetworkName {
			iface := map[string]interface{}{
				"name": network.Name,
			}
			
			// Add interface type specific configuration
			switch network.Type {
			case models.InternalNetworkTypeBridge:
				iface["bridge"] = map[string]interface{}{}
			case models.InternalNetworkTypeSRIOV:
				iface["sriov"] = map[string]interface{}{}
			default:
				iface["masquerade"] = map[string]interface{}{}
			}
			
			interfaces = append(interfaces, iface)
		}
	}
	
	return interfaces
}
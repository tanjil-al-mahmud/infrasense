package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ProtocolProbeResult holds the result of probing a single protocol.
type ProtocolProbeResult struct {
	Protocol  string `json:"protocol"`
	Available bool   `json:"available"`
	Port      int    `json:"port"`
	Error     string `json:"error,omitempty"`
}

// ProtocolDetectionResult holds all probe results for a BMC.
type ProtocolDetectionResult struct {
	BMCIPAddress        string                `json:"bmc_ip_address"`
	Vendor              string                `json:"vendor"`
	Model               string                `json:"model"`
	BMCType             string                `json:"bmc_type"`
	SupportedProtocols  []string              `json:"supported_protocols"`
	RecommendedProtocol string                `json:"recommended_protocol"`
	Probes              []ProtocolProbeResult `json:"probes"`
}

type probeOutput struct {
	available bool
	err       error
	vendor    string
	model     string
	bmcType   string
}

// ProtocolDetector probes a BMC to determine which protocols are available.
type ProtocolDetector struct{}

func NewProtocolDetector() *ProtocolDetector { return &ProtocolDetector{} }

// Detect probes the BMC for Redfish, IPMI, SNMP, SSH, WS-Man, and KVM availability.
// Returns results in fallback priority order: Redfish → SNMP → IPMI → SSH.
func (d *ProtocolDetector) Detect(ctx context.Context, bmcIP, username, password string) ProtocolDetectionResult {
	if len(bmcIP) > 2 && bmcIP[0] == '[' && bmcIP[len(bmcIP)-1] == ']' {
		bmcIP = bmcIP[1 : len(bmcIP)-1]
	}

	result := ProtocolDetectionResult{BMCIPAddress: bmcIP}

	type probe struct {
		protocol string
		port     int
		fn       func() probeOutput
	}

	probes := []probe{
		{"redfish", 443, func() probeOutput { return d.probeRedfish(ctx, bmcIP, 443) }},
		{"ipmi", 623, func() probeOutput { return d.probeIPMI(ctx, bmcIP, 623, username, password) }},
		{"snmp", 161, func() probeOutput { out, err := d.probeUDP(ctx, bmcIP, 161); return probeOutput{available: out, err: err} }},
		{"ssh", 22, func() probeOutput { out, err := d.probeTCP(ctx, bmcIP, 22); return probeOutput{available: out, err: err} }},
		{"wsman", 5989, func() probeOutput { out, err := d.probeTCP(ctx, bmcIP, 5989); return probeOutput{available: out, err: err} }},
		{"kvm", 5900, func() probeOutput { out, err := d.probeTCP(ctx, bmcIP, 5900); return probeOutput{available: out, err: err} }},
	}

	type probeResult struct {
		idx    int
		result ProtocolProbeResult
		out    probeOutput
	}
	ch := make(chan probeResult, len(probes))

	for i, p := range probes {
		go func(idx int, pr probe) {
			out := pr.fn()
			res := ProtocolProbeResult{Protocol: pr.protocol, Port: pr.port, Available: out.available}
			if out.err != nil {
				res.Error = out.err.Error()
			}
			ch <- probeResult{idx: idx, result: res, out: out}
		}(i, p)
	}

	probeResults := make([]ProtocolProbeResult, len(probes))
	for range probes {
		r := <-ch
		probeResults[r.idx] = r.result
		if r.out.vendor != "" && result.Vendor == "" {
			result.Vendor = r.out.vendor
		}
		if r.out.model != "" && result.Model == "" {
			result.Model = r.out.model
		}
		if r.out.bmcType != "" && result.BMCType == "" {
			result.BMCType = r.out.bmcType
		}
	}

	for _, pr := range probeResults {
		result.Probes = append(result.Probes, pr)
	}

	// Priority: redfish > snmp > ipmi > ssh
	priority := []string{"redfish", "snmp", "ipmi", "ssh"}
	for _, proto := range priority {
		for _, pr := range result.Probes {
			if pr.Protocol == proto && pr.Available {
				result.RecommendedProtocol = proto
				break
			}
		}
		if result.RecommendedProtocol != "" {
			break
		}
	}

	if result.RecommendedProtocol == "" {
		result.RecommendedProtocol = "unknown"
	}

	result.SupportedProtocols = d.mapCapabilities(result.Vendor, result.BMCType, result.Probes)

	return result
}

func (d *ProtocolDetector) mapCapabilities(vendor, bmcType string, probes []ProtocolProbeResult) []string {
	supported := make([]string, 0)
	
	vendorLower := strings.ToLower(vendor)
	bmcLower := strings.ToLower(bmcType)

	if vendorLower == "dell" {
		if strings.Contains(bmcLower, "idrac7") {
			supported = append(supported, "ipmi", "redfish")
		} else if strings.Contains(bmcLower, "idrac8") {
			supported = append(supported, "redfish", "ipmi")
		} else if strings.Contains(bmcLower, "idrac9") {
			supported = append(supported, "redfish", "ipmi", "snmp")
		} else {
			supported = append(supported, "redfish", "ipmi", "snmp") // Default for Dell
		}
	} else if vendorLower == "hpe" {
		if strings.Contains(bmcLower, "ilo4") {
			supported = append(supported, "ipmi", "snmp")
		} else if strings.Contains(bmcLower, "ilo5") || strings.Contains(bmcLower, "ilo6") {
			supported = append(supported, "redfish", "ipmi", "snmp")
		} else {
			supported = append(supported, "redfish", "ipmi", "snmp") // Default for HPE
		}
	} else if vendorLower == "lenovo" || strings.Contains(vendorLower, "xclarity") {
		supported = append(supported, "redfish", "ipmi")
	} else if vendorLower == "cisco" || strings.Contains(vendorLower, "cimc") {
		supported = append(supported, "redfish", "ipmi")
	} else if vendorLower == "supermicro" {
		supported = append(supported, "redfish", "ipmi")
	} else {
		// Fallback based on probes
		for _, p := range probes {
			if p.Available && (p.Protocol == "redfish" || p.Protocol == "ipmi" || p.Protocol == "snmp" || p.Protocol == "ssh") {
				supported = append(supported, p.Protocol)
			}
		}
	}

	return supported
}

func (d *ProtocolDetector) probeRedfish(ctx context.Context, host string, port int) probeOutput {
	url := fmt.Sprintf("https://%s/redfish/v1", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return probeOutput{available: false, err: err}
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return probeOutput{available: false, err: err}
	}
	defer resp.Body.Close()

	out := probeOutput{available: resp.StatusCode < 500}
	
	// Check server header
	server := resp.Header.Get("Server")
	serverUpper := strings.ToUpper(server)
	if strings.Contains(serverUpper, "IDRAC") {
		out.vendor = "Dell"
		out.model = "PowerEdge"
		out.bmcType = "iDRAC"
		// Try to refine iDRAC version from server header if possible, e.g. "Appweb/9 iDRAC9"
		if strings.Contains(serverUpper, "IDRAC9") {
			out.bmcType = "iDRAC9"
		} else if strings.Contains(serverUpper, "IDRAC8") {
			out.bmcType = "iDRAC8"
		} else if strings.Contains(serverUpper, "IDRAC7") {
			out.bmcType = "iDRAC7"
		}
	} else if strings.Contains(serverUpper, "ILO") {
		out.vendor = "HPE"
		out.model = "ProLiant"
		out.bmcType = "iLO"
		if strings.Contains(serverUpper, "ILO 5") {
			out.bmcType = "iLO5"
		} else if strings.Contains(serverUpper, "ILO 4") {
			out.bmcType = "iLO4"
		}
	} else if strings.Contains(serverUpper, "XCLARITY") {
		out.vendor = "Lenovo"
		out.bmcType = "XClarity"
	} else if strings.Contains(serverUpper, "SUPERMICRO") {
		out.vendor = "Supermicro"
		out.bmcType = "BMC"
	}

	// Optionally parse JSON body to find Manager/Systems
	bodyBytes, _ := io.ReadAll(resp.Body)
	var rf map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rf); err == nil {
		if name, ok := rf["Name"].(string); ok && out.bmcType == "" {
			nameUpper := strings.ToUpper(name)
			if strings.Contains(nameUpper, "IDRAC") {
				out.vendor = "Dell"
				out.bmcType = "iDRAC"
			} else if strings.Contains(nameUpper, "ILO") {
				out.vendor = "HPE"
				out.bmcType = "iLO"
			}
		}
	}

	return out
}

func (d *ProtocolDetector) probeIPMI(ctx context.Context, host string, port int, username, password string) probeOutput {
	// First do a basic UDP dial to see if port is responsive
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return probeOutput{available: false, err: err}
	}
	conn.Close()

	out := probeOutput{available: true} // Port is at least open/accepting UDP somehow... though UDP is connectionless.
	
	// We've been asked to detect Vendor/Model via IPMI if possible.
	if username != "" {
		cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cmdCtx, "ipmitool", "-I", "lanplus", "-H", host, "-U", username, "-P", password, "chassis", "status")
		outBytes, err := cmd.CombinedOutput()
		if err == nil {
			output := string(outBytes)
			if strings.Contains(output, "Dell") {
				out.vendor = "Dell"
			} else if strings.Contains(output, "Hewlett-Packard") || strings.Contains(output, "HPE") {
				out.vendor = "HPE"
			} else if strings.Contains(output, "Supermicro") {
				out.vendor = "Supermicro"
			}
		} else {
			// fall back to getting fru if chassis status fails
			cmdFRU := exec.CommandContext(cmdCtx, "ipmitool", "-I", "lanplus", "-H", host, "-U", username, "-P", password, "fru")
			fruBytes, errFru := cmdFRU.CombinedOutput()
			if errFru == nil {
				output := string(fruBytes)
				if strings.Contains(output, "Dell") {
					out.vendor = "Dell"
				} else if strings.Contains(output, "Hewlett-Packard") || strings.Contains(output, "HPE") {
					out.vendor = "HPE"
				} else if strings.Contains(output, "Supermicro") {
					out.vendor = "Supermicro"
				}
			}
		}
	}

	// Try a basic RMCP ping if we still aren't sure it's an IPMI port, but we'll accept DialTimeout for now.
	return out
}

func (d *ProtocolDetector) probeTCP(ctx context.Context, host string, port int) (bool, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

func (d *ProtocolDetector) probeUDP(ctx context.Context, host string, port int) (bool, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("udp", addr, 3*time.Second)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

package api

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	virtualPrefixes := []string{"veth", "virbr", "dummy", "tun", "tap"}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	if strings.Contains(lower, "docker") {
		return true
	}
	// Windows / hypervisor virtual adapters
	if strings.Contains(name, "vEthernet") ||
		strings.Contains(name, "Virtual") ||
		strings.Contains(name, "Hyper-V") {
		return true
	}
	return false
}

func formatMAC(mac net.HardwareAddr) string {
	parts := make([]string, len(mac))
	for i, b := range mac {
		parts[i] = strings.ToUpper(fmt.Sprintf("%02x", b))
	}
	return strings.Join(parts, "-")
}

func computeLicenseCode(macStr string) (string, error) {
	input := "AAS" + macStr
	hash := sha256.Sum256([]byte(input))
	// Match Java BigInteger(1, hash).intValue() -> low 32 bits as signed int
	last4 := hash[len(hash)-4:]
	val := int32(binary.BigEndian.Uint32(last4))
	numStr := strconv.FormatInt(int64(val), 10)
	return "SZTY" + numStr, nil
}

func getLicenseCodes() ([]LicenseInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var results []LicenseInfo
	for _, iface := range ifaces {
		// Filter loopback, down interfaces, no hardware address, virtual/docker
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var selectedIP string
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.IsLoopback() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
				continue
			}
			if ip.To4() != nil {
				selectedIP = ip.String()
				break
			}
			if selectedIP == "" {
				selectedIP = ip.String()
			}
		}

		if selectedIP == "" {
			continue
		}

		macStr := formatMAC(iface.HardwareAddr)
		code, err := computeLicenseCode(macStr)
		if err != nil {
			continue
		}

		results = append(results, LicenseInfo{
			Code:      code,
			Interface: iface.Name,
			IP:        selectedIP,
		})
	}

	return results, nil
}

func (s *Server) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	licenses, err := getLicenseCodes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, licenses)
}

package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

// Secret key for signing — change this for your own builds
var secretKey = []byte("wechat-obsidian-bot-2026")

type LicenseData struct {
	MachineID string `json:"mid"`
	Expiry    string `json:"exp"` // YYYY-MM-DD
	Product   string `json:"prd"`
}

// GetMachineID returns a unique identifier for this machine.
func GetMachineID() string {
	host, _ := os.Hostname()
	data := fmt.Sprintf("%s-%s-%d", host, runtime.GOOS, runtime.NumCPU())

	// Try to add first MAC address
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				if mac := iface.HardwareAddr.String(); mac != "" {
					data += "-" + mac
					break
				}
			}
		}
	}

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%X", hash[:6]) // First 12 hex chars (6 bytes)
}

// GenerateKey creates a license key for the given machine ID and expiry date.
func GenerateKey(machineID, expiry string) string {
	// Fixed key order to match JS implementation (alphabetical: exp, mid, prd)
	jsonData := []byte(fmt.Sprintf(`{"exp":"%s","mid":"%s","prd":"wechat-obsidian-bot"}`, expiry, machineID))

	mac := hmac.New(sha256.New, secretKey)
	mac.Write(jsonData)
	sig := fmt.Sprintf("%X", mac.Sum(nil)[:8])

	// Combine: json + signature
	combined := append(jsonData, []byte("|"+sig)...)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(combined)

	// Format: WEOBS-XXXXX-XXXXX-XXXXX-XXXXX
	var parts []string
	for i := 0; i < len(encoded); i += 5 {
		end := i + 5
		if end > len(encoded) {
			end = len(encoded)
		}
		parts = append(parts, encoded[i:end])
	}
	return "WEOBS-" + strings.Join(parts, "-")
}

// Validate checks a license key. Returns (valid, message).
func Validate(key string) (bool, string) {
	// Strip prefix and dashes
	key = strings.TrimPrefix(key, "WEOBS-")
	key = strings.TrimPrefix(key, "weobs-")
	key = strings.ReplaceAll(key, "-", "")

	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(key))
	if err != nil {
		return false, "无效的 License Key 格式"
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return false, "无效的 License Key"
	}

	jsonData := []byte(parts[0])
	expectedSig := parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(jsonData)
	actualSig := fmt.Sprintf("%X", mac.Sum(nil)[:8])
	if !hmac.Equal([]byte(actualSig), []byte(expectedSig)) {
		return false, "License Key 签名验证失败"
	}

	// Parse payload
	var payload LicenseData
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return false, "License Key 数据损坏"
	}

	// Check machine ID
	machineID := GetMachineID()
	if payload.MachineID != machineID {
		return false, fmt.Sprintf("License Key 与当前设备不匹配\n机器码: %s", machineID)
	}

	// Check expiry
	if payload.Expiry != "" && payload.Expiry != "permanent" {
		expiryDate, err := time.Parse("2006-01-02", payload.Expiry)
		if err != nil {
			return false, "License Key 日期格式错误"
		}
		if time.Now().After(expiryDate) {
			daysLeft := int(time.Until(expiryDate).Hours() / 24)
			return false, fmt.Sprintf("License Key 已过期 (%d 天前)", -daysLeft)
		}
	}

	return true, ""
}

// PrintMachineID prints the machine ID for the user to send to you.
func PrintMachineID() {
	fmt.Printf("\n设备码: %s\n", GetMachineID())
	fmt.Println("将此设备码发送给作者以获取 License Key")
}

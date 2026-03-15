package collector

import (
	"testing"
)

func TestParseFRUOutput(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantManufacturer string
		wantProductName  string
		wantSerialNumber string
	}{
		{
			name: "standard Dell FRU output",
			input: `FRU Device Description : Builtin FRU Device (ID 0)
 Board Mfg Date        : Mon Jan  1 00:00:00 1996
 Board Mfg             : Dell Inc.
 Board Product         : PowerEdge R740
 Board Serial          : BRDSERIAL
 Board Part Number     : 0ABC123
 Product Manufacturer  : Dell Inc.
 Product Name          : PowerEdge R740
 Product Part Number   : 0XYZ789
 Product Version       : A00
 Product Serial        : ABC1234
 Product Asset Tag     : ASSET001
`,
			wantManufacturer: "Dell Inc.",
			wantProductName:  "PowerEdge R740",
			wantSerialNumber: "ABC1234",
		},
		{
			name: "HPE server FRU output",
			input: `FRU Device Description : Builtin FRU Device (ID 0)
 Product Manufacturer  : HPE
 Product Name          : ProLiant DL380 Gen10
 Product Serial        : USE1234567
`,
			wantManufacturer: "HPE",
			wantProductName:  "ProLiant DL380 Gen10",
			wantSerialNumber: "USE1234567",
		},
		{
			name:             "empty output",
			input:            "",
			wantManufacturer: "",
			wantProductName:  "",
			wantSerialNumber: "",
		},
		{
			name: "missing serial number",
			input: `Product Manufacturer  : Supermicro
 Product Name          : X11DPi-N
`,
			wantManufacturer: "Supermicro",
			wantProductName:  "X11DPi-N",
			wantSerialNumber: "",
		},
		{
			name: "value with colon in it",
			input: `Product Manufacturer  : Dell Inc.
 Product Name          : PowerEdge R740
 Product Serial        : SN:ABC:1234
`,
			wantManufacturer: "Dell Inc.",
			wantProductName:  "PowerEdge R740",
			wantSerialNumber: "SN:ABC:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manufacturer, productName, serialNumber := parseFRUOutput(tt.input)
			if manufacturer != tt.wantManufacturer {
				t.Errorf("manufacturer = %q, want %q", manufacturer, tt.wantManufacturer)
			}
			if productName != tt.wantProductName {
				t.Errorf("productName = %q, want %q", productName, tt.wantProductName)
			}
			if serialNumber != tt.wantSerialNumber {
				t.Errorf("serialNumber = %q, want %q", serialNumber, tt.wantSerialNumber)
			}
		})
	}
}

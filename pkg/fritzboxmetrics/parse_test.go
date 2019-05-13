package fritzboxmetrics

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testfile(name string) *os.File {
	file, err := os.Open(fmt.Sprintf("testdata/%s", name))
	if err != nil {
		panic(err)
	}
	return file
}

func rootTestfile(t *testing.T, name string) (root Root) {
	file := testfile(name)
	defer file.Close()

	dec := xml.NewDecoder(file)
	err := dec.Decode(&root)
	require.NoError(t, err)

	return
}

func TestParseIgddesc(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	root := rootTestfile(t, "igddesc.xml")

	// Root level
	rootDevice := root.Device
	assert.Equal("urn:schemas-upnp-org:device:InternetGatewayDevice:1", rootDevice.DeviceType)
	assert.Equal("FRITZ!Box Fon WLAN 7360", rootDevice.FriendlyName)
	assert.Equal("AVM Berlin", rootDevice.Manufacturer)
	assert.Equal("", rootDevice.ManufacturerURL)
	assert.Equal("FRITZ!Box Fon WLAN 7360", rootDevice.ModelDescription)
	assert.Equal("FRITZ!Box Fon WLAN 7360", rootDevice.ModelName)
	assert.Equal("avme", rootDevice.ModelNumber)
	assert.Equal("", rootDevice.ModelURL)
	assert.Equal("uuid:75802409-bccb-40e7-8e6c-0123456789AB", rootDevice.UDN)

	// Root level services
	require.Len(root.Device.Services, 1)
	service := root.Device.Services[0]
	assert.Equal("urn:schemas-any-com:service:Any:1", service.ServiceType)
	assert.Equal("urn:any-com:serviceId:any1", service.ServiceID)
	assert.Equal("/igdupnp/control/any", service.ControlURL)
	assert.Equal("/igdupnp/control/any", service.EventSubURL)
	assert.Equal("/any.xml", service.SCPDURL)

	// Second level devices
	require.Len(root.Device.Devices, 1)
	device := root.Device.Devices[0]
	assert.Equal("urn:schemas-upnp-org:device:WANDevice:1", device.DeviceType)
	assert.Equal("WANDevice - FRITZ!Box Fon WLAN 7360", device.FriendlyName)
	assert.Equal("AVM Berlin", device.Manufacturer)
	assert.Equal("", device.ManufacturerURL)
	assert.Equal("WANDevice - FRITZ!Box Fon WLAN 7360", device.ModelDescription)
	assert.Equal("WANDevice - FRITZ!Box Fon WLAN 7360", device.ModelName)
	assert.Equal("avme", device.ModelNumber)
	assert.Equal("", device.ModelURL)
	assert.Equal("uuid:76802409-bccb-40e7-8e6b-0123456789AB", device.UDN)

	// Second level services
	services := device.Services
	require.Len(services, 1)
	service = services[0]

	assert.Equal("urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1", service.ServiceType)
	assert.Equal("urn:upnp-org:serviceId:WANCommonIFC1", service.ServiceID)
	assert.Equal("/igdupnp/control/WANCommonIFC1", service.ControlURL)
	assert.Equal("/igdupnp/control/WANCommonIFC1", service.EventSubURL)
	assert.Equal("/igdicfgSCPD.xml", service.SCPDURL)

	// Third level device
	require.Len(device.Devices, 1)
	device = device.Devices[0]

	assert.Equal("urn:schemas-upnp-org:device:WANConnectionDevice:1", device.DeviceType)
	assert.Equal("WANConnectionDevice - FRITZ!Box Fon WLAN 7360", device.FriendlyName)
	assert.Equal("AVM Berlin", device.Manufacturer)
	assert.Equal("", device.ManufacturerURL)
	assert.Equal("WANConnectionDevice - FRITZ!Box Fon WLAN 7360", device.ModelDescription)
	assert.Equal("WANConnectionDevice - FRITZ!Box Fon WLAN 7360", device.ModelName)
	assert.Equal("avme", device.ModelNumber)
	assert.Equal("", device.ModelURL)
	assert.Equal("uuid:76802409-bccb-40e7-8e6a-0123456789AB", device.UDN)

	// Third level services
	services = device.Services
	require.Len(services, 3)
	service = services[0]
	assert.Equal("urn:schemas-upnp-org:service:WANDSLLinkConfig:1", service.ServiceType)
	assert.Equal("urn:upnp-org:serviceId:WANDSLLinkC1", service.ServiceID)
	assert.Equal("/igdupnp/control/WANDSLLinkC1", service.ControlURL)
	assert.Equal("/igdupnp/control/WANDSLLinkC1", service.EventSubURL)
	assert.Equal("/igddslSCPD.xml", service.SCPDURL)
}

func TestParseTr64desc(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	root := rootTestfile(t, "tr64desc.xml")

	// Root level
	rootDevice := root.Device
	assert.Equal("urn:dslforum-org:device:InternetGatewayDevice:1", rootDevice.DeviceType)
	assert.Equal("FRITZ!Box Fon WLAN 7360", rootDevice.FriendlyName)
	assert.Equal("AVM", rootDevice.Manufacturer)

	devices := root.Device.Devices
	require.Len(devices, 2)

	assert.Equal("LANDevice - FRITZ!Box Fon WLAN 7360", devices[0].ModelName)
	assert.Len(devices[0].Services, 5)
	assert.Len(devices[0].Devices, 0)

	assert.Equal("WANDevice - FRITZ!Box Fon WLAN 7360", devices[1].ModelName)
	assert.Len(devices[1].Services, 2)
	require.Len(devices[1].Devices, 1)
	assert.Len(devices[1].Devices[0].Services, 4)
}

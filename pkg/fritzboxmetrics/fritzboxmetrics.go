// Package fritzboxmetrics provides metrics fro the UPnP and Tr64 interface
package fritzboxmetrics

// Copyright 2016 Nils Decker
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// curl http://fritz.box:49000/igddesc.xml
// curl http://fritz.box:49000/any.xml
// curl http://fritz.box:49000/igdconnSCPD.xml
// curl http://fritz.box:49000/igdicfgSCPD.xml
// curl http://fritz.box:49000/igddslSCPD.xml
// curl http://fritz.box:49000/igd2ipv6fwcSCPD.xml

const textXML = `text/xml; charset="utf-8"`

// Root of the UPNP tree
type Root struct {
	BaseURL  string
	Username string
	Password string
	Device   Device              `xml:"device"`
	Services map[string]*Service // Map of all services indexed by .ServiceType
}

// Device represents an UPNP device
type Device struct {
	root *Root

	DeviceType       string `xml:"deviceType"`
	FriendlyName     string `xml:"friendlyName"`
	Manufacturer     string `xml:"manufacturer"`
	ManufacturerURL  string `xml:"ManufacturerURL"`
	ModelDescription string `xml:"modelDescription"`
	ModelName        string `xml:"modelName"`
	ModelNumber      string `xml:"modelNumber"`
	ModelURL         string `xml:"ModelURL"`
	UDN              string `xml:"UDN"`

	Services []*Service `xml:"serviceList>service"` // Service of the device
	Devices  []*Device  `xml:"deviceList>device"`   // Sub-Devices of the device

	PresentationURL string `xml:"PresentationURL"`
}

// Service represents an UPnP Service
type Service struct {
	Device *Device

	ServiceType string `xml:"serviceType"`
	ServiceID   string `xml:"serviceId"`
	ControlURL  string `xml:"controlURL"`
	EventSubURL string `xml:"eventSubURL"`
	SCPDURL     string `xml:"SCPDURL"`

	Actions        map[string]*Action // All actions available on the service
	StateVariables []*StateVariable   // All state variables available on the service
}

type scpdRoot struct {
	Actions        []*Action        `xml:"actionList>action"`
	StateVariables []*StateVariable `xml:"serviceStateTable>stateVariable"`
}

// Action represents an UPnP Action on a Service
type Action struct {
	service *Service

	Name        string               `xml:"name"`
	Arguments   []*Argument          `xml:"argumentList>argument"`
	ArgumentMap map[string]*Argument // Map of arguments indexed by .Name
}

// An Argument to an action
type Argument struct {
	Name                 string `xml:"name"`
	Direction            string `xml:"direction"`
	RelatedStateVariable string `xml:"relatedStateVariable"`
	StateVariable        *StateVariable
}

// StateVariable is a variable that can be manipulated through actions
type StateVariable struct {
	Name         string `xml:"name"`
	DataType     string `xml:"dataType"`
	DefaultValue string `xml:"defaultValue"`
}

// Result are all output argements of the Call():
// The map is indexed by the name of the state variable.
// The type of the value is string, uint64 or bool depending of the DataType of the variable.
type Result map[string]interface{}

func (r *Root) fetchAndDecode(path string) error {
	uri := fmt.Sprintf("%s/%s", r.BaseURL, path)
	response, err := http.Get(uri)
	if err != nil {
		return errors.Wrapf(err, "could not get %s", uri)
	}
	defer response.Body.Close()

	dec := xml.NewDecoder(response.Body)
	if err = dec.Decode(r); err != nil {
		return errors.Wrap(err, "could not decode XML")
	}

	return nil
}

// load the whole tree
func (r *Root) load() error {
	if err := r.fetchAndDecode("igddesc.xml"); err != nil {
		return err
	}

	r.Services = make(map[string]*Service)
	return r.Device.fillServices(r)
}

func (r *Root) loadTr64() error {
	if err := r.fetchAndDecode("tr64desc.xml"); err != nil {
		return err
	}

	r.Services = make(map[string]*Service)
	return r.Device.fillServices(r)
}

// load all service descriptions
func (d *Device) fillServices(r *Root) error {
	d.root = r

	for _, s := range d.Services {
		s.Device = d

		response, err := http.Get(r.BaseURL + s.SCPDURL)
		if err != nil {
			return errors.Wrap(err, "could not get service descriptions")
		}

		err = s.parseActions(response.Body)
		response.Body.Close()

		if err != nil {
			return err
		}

		r.Services[s.ServiceType] = s
	}

	// Handle sub-devices
	for _, d2 := range d.Devices {
		if err := d2.fillServices(r); err != nil {
			return errors.Wrap(err, "could not fill services")
		}
	}
	return nil
}

func (s *Service) parseActions(r io.Reader) error {
	var scpd scpdRoot

	dec := xml.NewDecoder(r)
	if err := dec.Decode(&scpd); err != nil {
		return errors.Wrap(err, "could not decode xml")
	}

	s.Actions = make(map[string]*Action)
	for _, a := range scpd.Actions {
		s.Actions[a.Name] = a
	}
	s.StateVariables = scpd.StateVariables

	for _, a := range s.Actions {
		a.service = s
		a.ArgumentMap = make(map[string]*Argument)

		for _, arg := range a.Arguments {
			for _, svar := range s.StateVariables {
				if arg.RelatedStateVariable == svar.Name {
					arg.StateVariable = svar
				}
			}

			a.ArgumentMap[arg.Name] = arg
		}
	}

	return nil
}

// LoadServices loads the services tree from a device.
func LoadServices(device string, port uint16, username string, password string) (*Root, error) {
	root := &Root{
		BaseURL:  fmt.Sprintf("http://%s:%d", device, port),
		Username: username,
		Password: password,
	}

	if err := root.load(); err != nil {
		return nil, errors.Wrap(err, "could not load root element")
	}

	rootTr64 := &Root{
		BaseURL:  fmt.Sprintf("http://%s:%d", device, port),
		Username: username,
		Password: password,
	}

	if err := rootTr64.loadTr64(); err != nil {
		return nil, errors.Wrap(err, "could not load Tr64")
	}

	for k, v := range rootTr64.Services {
		root.Services[k] = v
	}

	return root, nil
}

package fritzboxmetrics

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	dac "github.com/123Haynes/go-http-digest-auth-client"
	"github.com/pkg/errors"
)

// IsGetOnly returns if the action seems to be a query for information.
// This is determined by checking if the action has no input arguments and at least one output argument.
func (a *Action) IsGetOnly() bool {
	for _, a := range a.Arguments {
		if a.Direction == "in" {
			return false
		}
	}
	return len(a.Arguments) > 0
}

// Call an action.
// Currently only actions without input arguments are supported.
func (a *Action) Call() (Result, error) {
	bodystr := fmt.Sprintf(`
        <?xml version='1.0' encoding='utf-8'?>
        <s:Envelope s:encodingStyle='http://schemas.xmlsoap.org/soap/encoding/' xmlns:s='http://schemas.xmlsoap.org/soap/envelope/'>
            <s:Body>
                <u:%s xmlns:u='%s' />
            </s:Body>
        </s:Envelope>
    `, a.Name, a.service.ServiceType)

	url := a.service.Device.root.BaseURL + a.service.ControlURL
	body := strings.NewReader(bodystr)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new request")
	}

	action := fmt.Sprintf("%s#%s", a.service.ServiceType, a.Name)

	req.Header.Set("Content-Type", textXML)
	req.Header.Set("SoapAction", action)

	var resp *http.Response

	// Add digest authentication
	if username := a.service.Device.root.Username; username != "" {
		t := dac.NewTransport(username, a.service.Device.root.Password)
		resp, err = t.RoundTrip(req)

		if err != nil {
			return nil, errors.Wrap(err, "could not roundtrip digest authentication")
		}
	} else {
		resp, err = http.DefaultClient.Do(req)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ResponseError{
			URI:        url,
			StatusCode: resp.StatusCode,
		}
	}

	return a.parseSoapResponse(resp.Body)
}

func (a *Action) parseSoapResponse(r io.Reader) (Result, error) {
	res := make(Result)
	dec := xml.NewDecoder(r)

	for {
		t, err := dec.Token()
		if err == io.EOF {
			return res, nil
		}

		if err != nil {
			return nil, err
		}

		if se, ok := t.(xml.StartElement); ok {
			arg, ok := a.ArgumentMap[se.Name.Local]

			if ok {
				t2, err := dec.Token()
				if err != nil {
					return nil, err
				}

				var val string
				switch element := t2.(type) {
				case xml.EndElement:
					val = ""
				case xml.CharData:
					val = string(element)
				default:
					return nil, ErrInvalidSOAPResponse
				}

				converted, err := convertResult(val, arg)
				if err != nil {
					return nil, err
				}
				res[arg.StateVariable.Name] = converted
			}
		}

	}
}

func convertResult(val string, arg *Argument) (interface{}, error) {
	switch arg.StateVariable.DataType {
	case "string":
		return val, nil
	case "boolean":
		return bool(val == "1"), nil

	case "ui1", "ui2", "ui4":
		// type ui4 can contain values greater than 2^32!
		res, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "could nto parse uint")
		}
		return uint64(res), nil
	default:
		return nil, fmt.Errorf("unknown datatype: %s", arg.StateVariable.DataType)
	}
}

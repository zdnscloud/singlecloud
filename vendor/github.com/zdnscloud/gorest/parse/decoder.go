package parse

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/zdnscloud/gorest/parse/yaml"
)

type Decode func(interface{}) error

func GetDecoder(req *http.Request, reader io.Reader) Decode {
	if req.Header.Get("Content-type") == "application/yaml" {
		return yaml.NewYAMLToJSONDecoder(reader).Decode
	}
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return decoder.Decode
}

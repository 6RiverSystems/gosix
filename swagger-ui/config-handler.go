package swaggerui

import (
	"encoding/json"
	"net/http"
)

const ConfigLoadingPath = "/oas-ui-config"

// setting anything except url(s) from the dynamic config may not work, see
// upstream bug https://github.com/swagger-api/swagger-ui/issues/4455
var defaultConfig json.RawMessage = ([]byte)(`{
	"url": "../oas/openapi.yaml",
	"dom_id": "#swagger-ui",
	"deepLinking": true,
	"layout": "StandaloneLayout"
}`)

func DefaultConfigHandler() http.HandlerFunc {
	return CustomConfigHandler(func(config map[string]interface{}) map[string]interface{} { return config })
}

func CustomConfigHandler(
	customizer func(map[string]interface{}) map[string]interface{},
) http.HandlerFunc {
	var config map[string]interface{}
	var err error
	if err = json.Unmarshal(defaultConfig, &config); err != nil {
		// this should really be a compile error
		panic(err)
	}
	config = customizer(config)
	var msg []byte
	if msg, err = json.Marshal(config); err != nil {
		panic(err)
	}
	return func(writer http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		writer.Header().Add("Content-type", "application/json; charset=utf-8")
		if _, err := writer.Write(msg); err != nil {
			// this assumes we have recovery middleware
			panic(err)
		}
	}
}

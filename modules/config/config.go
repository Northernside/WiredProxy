package config

// JSON based configuration file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"wired.rip/wiredutils/protocol"
	"wired.rip/wiredutils/utils"
)

// Configuration struct
type RoutesConfig struct {
	Routes []protocol.Route `json:"routes"`
}

type SystemConfig struct {
	WiredHost string           `json:"wired_host"`
	SystemKey string           `json:"system_key"`
	Routes    []protocol.Route `json:"routes"`
}

var config SystemConfig

func AddRoute(route protocol.Route) int {
	config.Routes = append(config.Routes, route)
	saveConfigFile("config.json")

	return http.StatusOK
}

func SetRoutes(routes []protocol.Route) {
	config.Routes = routes
	saveConfigFile("config.json")
}

func DeleteRoute(routeId string) int {
	for i, r := range config.Routes {
		if r.RouteId == routeId {
			config.Routes = append(config.Routes[:i], config.Routes[i+1:]...)
			saveConfigFile("config.json")
			return http.StatusOK
		}
	}

	return http.StatusNotFound
}

func GetRoutes() []protocol.Route {
	return config.Routes
}

func GetSystemKey() string {
	return config.SystemKey
}

func GetWiredHost() string {
	return config.WiredHost
}

func GetRouteByProxyDomain(proxyDomain string) (protocol.Route, bool) {
	for _, r := range config.Routes {
		if r.ProxyDomain == proxyDomain {
			return r, true
		}
	}

	return protocol.Route{}, false
}

func Init() {
	// create if not exists
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		config = SystemConfig{
			WiredHost: "wired.rip",
			SystemKey: fmt.Sprintf("node-%s", utils.GenerateString(8)),
			Routes:    []protocol.Route{},
		}

		saveConfigFile("config.json")
	}

	config = readConfigFile("config.json")
}

func readConfigFile(configFile string) SystemConfig {
	var config SystemConfig

	file, err := os.Open(configFile)
	if err != nil {
		fmt.Println("Error opening configuration file:", err)
		return config
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading configuration file:", err)
		return config
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error parsing configuration file:", err)
		return config
	}

	return config
}

func saveConfigFile(configFile string) {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling configuration file:", err)
		return
	}

	err = ioutil.WriteFile(configFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing configuration file:", err)
	}
}

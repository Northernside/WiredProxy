package config

// JSON based configuration file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// Route
type Route struct {
	RouteId     string `json:"route_id"`
	ServerHost  string `json:"server_host"`
	ServerPort  string `json:"server_port"`
	ProxyDomain string `json:"proxy_domain"`
	ProxyPort   string `json:"proxy_port"`
}

// Configuration struct
type RoutesConfig struct {
	Routes []Route `json:"routes"`
}

var config RoutesConfig

func AddRoute(route Route) int {
	// check if proxy_port is already in use
	for _, r := range config.Routes {
		if r.ProxyPort == route.ProxyPort {
			return http.StatusConflict
		}
	}

	config.Routes = append(config.Routes, route)
	saveConfigFile("config.json")

	return http.StatusOK
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

func GetRoutes() []Route {
	return config.Routes
}

func GetRouteByProxyDomain(proxyDomain string) (Route, bool) {
	for _, r := range config.Routes {
		if r.ProxyDomain == proxyDomain {
			return r, true
		}
	}

	return Route{}, false
}

func init() {
	config = readConfigFile("config.json")
}

func readConfigFile(configFile string) RoutesConfig {
	var config RoutesConfig

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

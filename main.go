package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"net/http"

	"github.com/gin-gonic/gin"
)

type VM struct {
	Name   string `json:"Name"`
	Status string `json:"Status"`
}

type Host struct {
	Hostname string `json:"Hostname"`
	IP       string `json:"IP"`
	Port     string `json:"Port"`
	Username string `json:"Username"`
	Password string `json:"Password"`
	VMs      []VM   `json:"VMs"`
}

type Hosts struct {
	Hosts []Host `json:"hosts"`
}

func getHosts() []Host {
	// read hosts from json file
	file, _ := os.ReadFile("config.json")
	// log.Println(string(file))

	var hosts Hosts
	// unmarshal json to Host struct

	err := json.Unmarshal(file, &hosts)
	if err != nil {
		log.Fatalf("Error unmarshalling hosts: %v", err)
	}

	// log.Println(hosts)
	// get VMs for each host

	for i, host := range hosts.Hosts {
		hosts.Hosts[i].VMs = getVMs(host)
	}
	// log.Println(hosts.Hosts)

	return hosts.Hosts
}

func getVMs(host Host) []VM {
	// First call to get the token
	authURL := fmt.Sprintf("http://%s:%s/login", host.IP, host.Port)

	authPayload := map[string]string{
		"username": host.Username,
		"password": host.Password,
	}
	// log.Println("Auth URL:", authURL)
	authBody, _ := json.Marshal(authPayload)
	authResp, err := http.Post(authURL, "application/json", bytes.NewBuffer(authBody))
	if err != nil {
		fmt.Println("Error authenticating:", err)
		return nil
	}
	defer authResp.Body.Close()

	authData, _ := ioutil.ReadAll(authResp.Body)

	var authResult map[string]string
	json.Unmarshal(authData, &authResult)
	token := authResult["token"]

	// log.Println("Token:", token)

	// Second call to get the VMs
	vmsURL := fmt.Sprintf("http://%s:%s/api/vms", host.IP, host.Port)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", vmsURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	vmsResp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching VMs:", err)
		return nil
	}
	defer vmsResp.Body.Close()

	vmsData, _ := ioutil.ReadAll(vmsResp.Body)
	// log.Println("VMs:", string(vmsData))
	var vms []VM
	json.Unmarshal(vmsData, &vms)

	return vms
}

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/gethost", func(c *gin.Context) {
		hosts := getHosts()
		c.JSON(http.StatusOK, hosts)
	})

	r.POST("/getvms", func(c *gin.Context) {
		var host Host
		c.BindJSON(&host)
		vms := getVMs(host)
		c.JSON(http.StatusOK, vms)
	})

	r.Static("/static", "./web/static")
	r.Run(":8080")

}

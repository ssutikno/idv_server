package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

type VM struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Host struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	VMs  []VM   `json:"vms"`
}

var (
	hosts = make(map[string]Host)
	mu    sync.Mutex
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/hosts", createHost).Methods("POST")
	r.HandleFunc("/hosts/{id}", getHost).Methods("GET")
	r.HandleFunc("/hosts/{id}", updateHost).Methods("PUT")
	r.HandleFunc("/hosts/{id}", deleteHost).Methods("DELETE")
	r.HandleFunc("/hosts/{id}/vms", createVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}", getVM).Methods("GET")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/start", startVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/reboot", rebootVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/reset", resetVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/shutdown", shutdownVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/destroy", destroyVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}/copy", copyVM).Methods("POST")
	r.HandleFunc("/hosts/{hostId}/vms/{vmId}", deleteVM).Methods("DELETE")

	http.ListenAndServe(":8080", r)
}

func createHost(w http.ResponseWriter, r *http.Request) {
	var host Host
	if err := json.NewDecoder(r.Body).Decode(&host); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	hosts[host.ID] = host

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(host)
}

func getHost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID := vars["id"]

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(host)
}

func createVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID := vars["id"]

	var vm VM
	if err := json.NewDecoder(r.Body).Decode(&vm); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	host.VMs = append(host.VMs, vm)
	hosts[hostID] = host

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vm)
}

func getVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID := vars["hostId"]
	vmID := vars["vmId"]

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	for _, vm := range host.VMs {
		if vm.ID == vmID {
			json.NewEncoder(w).Encode(vm)
			return
		}
	}

	http.Error(w, "VM not found", http.StatusNotFound)
}

func startVM(w http.ResponseWriter, r *http.Request) {
	changeVMStatus(w, r, "started")
}

func rebootVM(w http.ResponseWriter, r *http.Request) {
	changeVMStatus(w, r, "rebooted")
}

func resetVM(w http.ResponseWriter, r *http.Request) {
	changeVMStatus(w, r, "reset")
}

func shutdownVM(w http.ResponseWriter, r *http.Request) {
	changeVMStatus(w, r, "shutdown")
}

func destroyVM(w http.ResponseWriter, r *http.Request) {
	changeVMStatus(w, r, "destroyed")
}

func copyVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID := vars["hostId"]
	vmID := vars["vmId"]

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	for _, vm := range host.VMs {
		if vm.ID == vmID {
			newVM := vm
			newVM.ID = vm.ID + "_copy"
			host.VMs = append(host.VMs, newVM)
			hosts[hostID] = host
			json.NewEncoder(w).Encode(newVM)
			return
		}
	}

	http.Error(w, "VM not found", http.StatusNotFound)
}

func deleteVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostID := vars["hostId"]
	vmID := vars["vmId"]

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	for i, vm := range host.VMs {
		if vm.ID == vmID {
			host.VMs = append(host.VMs[:i], host.VMs[i+1:]...)
			hosts[hostID] = host
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "VM not found", http.StatusNotFound)
}

func changeVMStatus(w http.ResponseWriter, r *http.Request, status string) {
	vars := mux.Vars(r)
	hostID := vars["hostId"]
	vmID := vars["vmId"]

	mu.Lock()
	defer mu.Unlock()
	host, exists := hosts[hostID]
	if !exists {
		http.Error(w, "Host not found", http.StatusNotFound)
		return
	}

	for i, vm := range host.VMs {
		if vm.ID == vmID {
			host.VMs[i].Status = status
			hosts[hostID] = host
			json.NewEncoder(w).Encode(host.VMs[i])
			return
		}
	}

	http.Error(w, "VM not found", http.StatusNotFound)
}
func distributeVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vmId"]
	targetHostID := vars["targetHostId"]

	mu.Lock()
	defer mu.Unlock()

	// Find the VM and its current host
	var currentHostID string
	var vm VM
	for hostID, host := range hosts {
		for i, v := range host.VMs {
			if v.ID == vmID {
				currentHostID = hostID
				vm = v
				// Remove VM from current host
				host.VMs = append(host.VMs[:i], host.VMs[i+1:]...)
				hosts[hostID] = host
				break
			}
		}
		if currentHostID != "" {
			break
		}
	}

	if currentHostID == "" {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	// Add VM to target host
	targetHost, exists := hosts[targetHostID]
	if !exists {
		http.Error(w, "Target host not found", http.StatusNotFound)
		return
	}

	targetHost.VMs = append(targetHost.VMs, vm)
	hosts[targetHostID] = targetHost

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(vm)
}
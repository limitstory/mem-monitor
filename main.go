package main

import (
	"context"
	"fmt"
	mod "mem_monitor/modules"
	"time"

	"github.com/shirou/gopsutil/mem"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	internalapi "k8s.io/cri-api/pkg/apis"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func GetSystemMemoryStatsInfo() mod.Memory {
	var get_memory mod.Memory

	memory, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}
	// fmt.Println(memory)

	get_memory.Total = memory.Total
	get_memory.Available = memory.Available
	get_memory.Used = memory.Total - memory.Available
	get_memory.UsedPercent = float64(get_memory.Used) / float64(memory.Total) * 100

	return get_memory
}

func GetPodInfo(client internalapi.RuntimeService) bool {

	var count int64 = 0

	filter := &pb.PodSandboxStatsFilter{}

	stats, err := client.ListPodSandboxStats(context.TODO(), filter)
	if err != nil {
		return true
	}

	for i := 0; i < len(stats); i++ {
		// Do not store namespaces other than default namespaces
		if stats[i].Attributes.Metadata.Namespace != "default" {
			continue
		}
		// Do not store info of notworking pods
		status, _ := client.PodSandboxStatus(context.TODO(), stats[i].Attributes.Id, false)
		if status.Status.State == 1 { // exception handling: SANDBOX_NOTREADY
			continue
		}
		count++
	}

	if count == 0 {
		return false
	} else {
		return true
	}
}

func main() {
	var minMemoryUsagePercent float64 = 100.0
	var maxMemoryUsagePercent float64 = 0
	var totalMemoryUsagePercent float64 = 0
	var iteration int64 = 0

	var getMemoryArray []float64

	const ENDPOINT string = "unix:///var/run/crio/crio.sock"
	client, err := remote.NewRemoteRuntimeService(ENDPOINT, time.Second*2, nil)
	if err != nil {
		panic(err)
	}

	for {
		if GetPodInfo(client) == false {
			break
		} else {
			get_memory := GetSystemMemoryStatsInfo()
			iteration++

			if minMemoryUsagePercent > get_memory.UsedPercent {
				minMemoryUsagePercent = get_memory.UsedPercent
			}
			if maxMemoryUsagePercent < get_memory.UsedPercent {
				maxMemoryUsagePercent = get_memory.UsedPercent
			}
			totalMemoryUsagePercent += get_memory.UsedPercent

			getMemoryArray = append(getMemoryArray, get_memory.UsedPercent)

			time.Sleep(time.Second)
		}
	}

	if iteration == 0 {
		fmt.Println("No Testing")
	} else {
		fmt.Println("minMemoryUsagePercent:", minMemoryUsagePercent)
		fmt.Println("averagetotalMemoryUsagePercent:", float64(totalMemoryUsagePercent)/float64(iteration))
		fmt.Println("maxMemoryUsagePercent:", maxMemoryUsagePercent)
		fmt.Println("getMemoryArray:", getMemoryArray)
	}
}

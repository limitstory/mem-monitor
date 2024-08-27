package main

import (
	"context"
	"fmt"
	mod "mem_monitor/modules"
	"os"
	"time"

	"github.com/shirou/gopsutil/mem"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	internalapi "k8s.io/cri-api/pkg/apis"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func IsSucceed(podsItems []v1.Pod) bool {
	for _, pod := range podsItems {
		if pod.Status.Phase != "Succeeded" {
			return false
		}
	}
	return true
}

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
		if status == nil { // exception handling: nil pointer
			continue
		}
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
	var pods *v1.PodList
	var err error
	var minMemoryUsagePercent float64 = 100.0
	var maxMemoryUsagePercent float64 = 0
	var totalMemoryUsagePercent float64 = 0
	var iteration int64 = 0

	var getMemoryArray []float64

	/*
		const ENDPOINT string = "unix:///var/run/crio/crio.sock"
		client, err := remote.NewRemoteRuntimeService(ENDPOINT, time.Second, nil)
		if err != nil {
			panic(err)
		}*/

	// kubernetes api 클라이언트 생성하는 모듈
	clientset := mod.InitClient()
	if clientset == nil {
		fmt.Println("Could not create client!")
		os.Exit(-1)
	}

	for {
		pods, err = clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}

		if IsSucceed(pods.Items) == false {
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
		} else {
			break
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

package main

import "github.com/NVIDIA/go-nvml/pkg/nvml"

func panicNVMLIf(p nvml.Return) {
	if p != nvml.SUCCESS {
		panic("nvml error: " + nvml.ErrorString(p))
	}
}

func panicNVMLT[T any](val T, p nvml.Return) T {
	panicNVMLIf(p)
	return val
}

func main() {
	panicNVMLIf(nvml.Init())
	defer func() {
		panicNVMLIf(nvml.Shutdown())
	}()
	devCnt := panicNVMLT(nvml.DeviceGetCount())
	println("device count ", devCnt)
}

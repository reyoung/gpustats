package main

import (
	"flag"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
	"time"
)

func panicNVMLIf(p nvml.Return) {
	if p != nvml.SUCCESS {
		panic("nvml error: " + nvml.ErrorString(p))
	}
}

func panicNVMLT[T any](val T, p nvml.Return) T {
	panicNVMLIf(p)
	return val
}

var (
	addr              = flag.String("addr", ":19300", "metric serving address")
	gUtilizationRates = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gpustats",
		Name:      "utilization_rates",
		Help:      "device utility rate",
	}, []string{"device_id", "type"})
	gMemInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gpustats",
		Name:      "memory_info",
		Help:      "device memory info",
	}, []string{"device_id", "type"})
	gPCIEThroughput = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gpustats",
		Name:      "pcie_throughput",
		Help:      "pci-e throughput",
	}, []string{"device_id", "type"})
)

func ignorePanic(fn func()) {
	defer func() {
		recover()
	}()
	fn()
}

func monitorDevice(devID int) {
	device := panicNVMLT(nvml.DeviceGetHandleByIndex(devID))
	devIDStr := strconv.Itoa(devID)
	for {
		utilizationRates := panicNVMLT(device.GetUtilizationRates())
		gUtilizationRates.WithLabelValues(devIDStr, "gpu").Set(float64(utilizationRates.Gpu))
		gUtilizationRates.WithLabelValues(devIDStr, "memory").Set(float64(utilizationRates.Memory))
		meminfo := panicNVMLT(device.GetMemoryInfo())
		gMemInfo.WithLabelValues(devIDStr, "used").Set(float64(meminfo.Used))
		gMemInfo.WithLabelValues(devIDStr, "freed").Set(float64(meminfo.Free))
		gMemInfo.WithLabelValues(devIDStr, "total").Set(float64(meminfo.Total))
		ignorePanic(func() {
			// Some system cannot read pci-e through put
			gPCIEThroughput.WithLabelValues(devIDStr, "tx").Set(float64(panicNVMLT(device.GetPcieThroughput(nvml.PCIE_UTIL_TX_BYTES))))
			gPCIEThroughput.WithLabelValues(devIDStr, "rx").Set(float64(panicNVMLT(device.GetPcieThroughput(nvml.PCIE_UTIL_RX_BYTES))))
			gPCIEThroughput.WithLabelValues(devIDStr, "count").Set(float64(panicNVMLT(device.GetPcieThroughput(nvml.PCIE_UTIL_COUNT))))
		})
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	flag.Parse()
	panicNVMLIf(nvml.Init())
	defer func() {
		panicNVMLIf(nvml.Shutdown())
	}()
	devCnt := panicNVMLT(nvml.DeviceGetCount())

	for i := 0; i < devCnt; i++ {
		go monitorDevice(i)
	}

	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(*addr, nil)
}

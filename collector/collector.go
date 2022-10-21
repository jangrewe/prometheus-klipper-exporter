package collector

import (
	"context"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type collector struct {
	ctx     context.Context
	target  string
	modules []string
	logger  log.Logger
}

func New(ctx context.Context, target string, modules []string, logger log.Logger) *collector {
	return &collector{ctx: ctx, target: target, modules: modules, logger: logger}
}

// Describe implements Prometheus.Collector.
func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

// Regex to match all non valid characters
var prometheusMetricNameInvalidCharactersRegex = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func getValidMetricName(str string) string {
	// convert hyphens to underscores and strip out all other invalid characters
	return prometheusMetricNameInvalidCharactersRegex.ReplaceAllString(strings.Replace(str, "-", "_", -1), "")
}

// Collect implements Prometheus.Collector.
func (c collector) Collect(ch chan<- prometheus.Metric) {

	// Process Stats (and Network Stats)
	if slices.Contains(c.modules, "process_stats") || slices.Contains(c.modules, "network_stats") {

		c.logger.Infof("Collecting process_stats for %s", c.target)

		result, err := c.fetchMoonrakerProcessStats(c.target)
		if err != nil {
			c.logger.Debug(err)
			return
		}

		if slices.Contains(c.modules, "process_stats") {
			memUnits := result.Result.MoonrakerStats[len(result.Result.MoonrakerStats)-1].MemUnits
			if memUnits != "kB" {
				c.logger.Errorf("Unexpected units %s for Moonraker memory usage", memUnits)
			} else {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_moonraker_memory_kb", "Moonraker memory usage in Kb.", nil, nil),
					prometheus.GaugeValue,
					float64(result.Result.MoonrakerStats[len(result.Result.MoonrakerStats)-1].Memory))
			}

			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_moonraker_cpu_usage", "Moonraker CPU usage.", nil, nil),
				prometheus.GaugeValue,
				result.Result.MoonrakerStats[len(result.Result.MoonrakerStats)-1].CpuUsage)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_moonraker_websocket_connections", "Moonraker Websocket connection count.", nil, nil),
				prometheus.GaugeValue,
				float64(result.Result.WebsocketConnections))
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_cpu_temp", "Klipper system CPU temperature in celsius.", nil, nil),
				prometheus.GaugeValue,
				result.Result.CpuTemp)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_cpu", "Klipper system CPU usage.", nil, nil),
				prometheus.GaugeValue,
				result.Result.SystemCpuUsage.Cpu)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_memory_total", "Klipper system total memory.", nil, nil),
				prometheus.GaugeValue,
				float64(result.Result.SystemMemory.Total))
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_memory_available", "Klipper system available memory.", nil, nil),
				prometheus.GaugeValue,
				float64(result.Result.SystemMemory.Available))
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_memory_used", "Klipper system used memory.", nil, nil),
				prometheus.GaugeValue,
				float64(result.Result.SystemMemory.Used))
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_system_uptime", "Klipper system uptime.", nil, nil),
				prometheus.CounterValue,
				result.Result.SystemUptime)
		}

		if slices.Contains(c.modules, "network_stats") {
			for key, element := range result.Result.Network {
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_rx_bytes", "Klipper network received bytes.", nil, nil),
					prometheus.CounterValue,
					float64(element.RxBytes))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_tx_bytes", "Klipper network transmitted bytes.", nil, nil),
					prometheus.CounterValue,
					float64(element.TxBytes))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_rx_packets", "Klipper network received packets.", nil, nil),
					prometheus.CounterValue,
					float64(element.RxPackets))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_tx_packets", "Klipper network transmitted packets.", nil, nil),
					prometheus.CounterValue,
					float64(element.TxPackets))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_rx_errs", "Klipper network received errored packets.", nil, nil),
					prometheus.CounterValue,
					float64(element.RxErrs))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_tx_errs", "Klipper network transmitted errored packets.", nil, nil),
					prometheus.CounterValue,
					float64(element.TxErrs))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_rx_drop", "Klipper network received dropped packets.", nil, nil),
					prometheus.CounterValue,
					float64(element.RxDrop))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_tx_drop", "Klipper network transmitted dropped packtes.", nil, nil),
					prometheus.CounterValue,
					float64(element.TxDrop))
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_network_"+key+"_bandwidth", "Klipper network bandwidth.", nil, nil),
					prometheus.GaugeValue,
					element.Bandwidth)
			}
		}
	}

	// Directory Information
	if slices.Contains(c.modules, "directory_info") {
		c.logger.Infof("Collecting directory_info for %s", c.target)
		result, _ := c.fetchMoonrakerDirectoryInfo(c.target)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_disk_usage_total", "Klipper total disk space.", nil, nil),
			prometheus.GaugeValue,
			float64(result.Result.DiskUsage.Total))
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_disk_usage_used", "Klipper used disk space.", nil, nil),
			prometheus.GaugeValue,
			float64(result.Result.DiskUsage.Used))
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_disk_usage_available", "Klipper available disk space.", nil, nil),
			prometheus.GaugeValue,
			float64(result.Result.DiskUsage.Free))
	}
	// Job Queue
	if slices.Contains(c.modules, "job_queue") {
		c.logger.Infof("Collecting job_queue for %s", c.target)
		result, _ := c.fetchMoonrakerJobQueue(c.target)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_job_queue_length", "Klipper job queue length.", nil, nil),
			prometheus.GaugeValue,
			float64(len(result.Result.QueuedJobs)))
	}

	// System Info
	if slices.Contains(c.modules, "system_info") {
		c.logger.Infof("Collecting system_info for %s", c.target)
		result, _ := c.fetchMoonrakerSystemInfo(c.target)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_system_cpu_count", "Klipper system CPU count.", nil, nil),
			prometheus.GaugeValue,
			float64(result.Result.SystemInfo.CpuInfo.CpuCount))
	}

	// Temperature Store
	if slices.Contains(c.modules, "temperature") {
		c.logger.Infof("Collecting system_info for %s", c.target)
		result, _ := c.fetchTemperatureData(c.target)

		for k, v := range result.Result {
			c.logger.Debug(k)
			item := strings.ReplaceAll(k, " ", "_")
			attributes := v.(map[string]interface{})
			for k1, v1 := range attributes {
				c.logger.Debug("  " + k1)
				values := v1.([]interface{})
				label := strings.ReplaceAll(k1[0:len(k1)-1], " ", "_")
				ch <- prometheus.MustNewConstMetric(
					prometheus.NewDesc("klipper_"+item+"_"+label, "Klipper "+k+" "+label, nil, nil),
					prometheus.GaugeValue,
					values[len(values)-1].(float64))
			}
		}
	}

	// Printer Objects
	if slices.Contains(c.modules, "printer_objects") {
		c.logger.Infof("Collecting printer_objects for %s", c.target)
		result, _ := c.fetchMoonrakerPrinterObjects(c.target)

		// gcode_move
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_gcode_speed_factor", "Klipper gcode speed factor.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.GcodeMove.SpeedFactor)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_gcode_speed", "Klipper gcode speed.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.GcodeMove.Speed)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_gcode_extrude_factor", "Klipper gcode extrude factor.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.GcodeMove.ExtrudeFactor)

		// toolhead
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_print_time", "Klipper toolhead print time.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.PrintTime)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_estimated_print_time", "Klipper estimated print time.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.EstimatedPrintTime)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_max_velocity", "Klipper toolhead max velocity.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.MaxVelocity)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_max_accel", "Klipper toolhead max acceleration.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.MaxAccel)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_max_accel_to_decel", "Klipper toolhead max acceleration to deceleration.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.MaxAccelToDecel)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_toolhead_square_corner_velocity", "Klipper toolhead square corner velocity.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Toolhead.SquareCornerVelocity)

		// extruder
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_extruder_temperature", "Klipper extruder temperature.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Extruder.Temperature)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_extruder_target", "Klipper extruder target.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Extruder.Target)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_extruder_power", "Klipper extruder power.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Extruder.Power)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_extruder_pressure_advance", "Klipper extruder pressure advance.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Extruder.PressureAdvance)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_extruder_smooth_time", "Klipper extruder smooth time.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Extruder.SmoothTime)

		// heater_bed
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_heater_bed_temperature", "Klipper heater bed temperature.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.HeaterBed.Temperature)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_heater_bed_target", "Klipper heater bed target.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.HeaterBed.Target)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_heater_bed_power", "Klipper heater bed power.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.HeaterBed.Power)

		// fan
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_fan_speed", "Klipper fan speed.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Fan.Speed)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_fan_rpm", "Klipper fan rpm.", nil, nil),
			prometheus.GaugeValue,
			result.Result.Status.Fan.Rpm)

		// idle_timeout
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_printing_time", "The amount of time the printer has been in the Printing state.", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.IdleTimeout.PrintingTime)

		// virtual_sdcard
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_file_progress", "The print progress reported as a percentage of the file read.", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.VirtualSdCard.Progress)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_file_position", "The current file position in bytes.", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.VirtualSdCard.FilePosition)

		// print_stats
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_total_duration", "The total time (in seconds) elapsed since a print has started.", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.PrintStats.TotalDuration)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_print_duration", "The total time spent printing (in seconds).", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.PrintStats.PrintDuration)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_filament_used", "The amount of filament used during the current print (in mm)..", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.PrintStats.FilamentUsed)

		// display_status
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("klipper_print_gcode_progress", "The percentage of print progress, as reported by M73.", nil, nil),
			prometheus.CounterValue,
			result.Result.Status.DisplayStatus.Progress)

		// temperature_sensor
		for sk, sv := range result.Result.Status.TemperatureSensors {
			metricName := getValidMetricName(sk)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_temperature", "The temperature of the "+sk+" temperature sensor", nil, nil),
				prometheus.GaugeValue,
				sv.Temperature)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_measured_min_temp", "The measured minimun temperature of the "+sk+" temperature sensor", nil, nil),
				prometheus.GaugeValue,
				sv.MeasuredMinTemp)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_measured_max_temp", "The measured maximum temperature of the "+sk+" temperature sensor", nil, nil),
				prometheus.GaugeValue,
				sv.MeasuredMaxTemp)
		}

		// temperature_fan
		for fk, fv := range result.Result.Status.TemperatureFans {
			metricName := getValidMetricName(fk)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_speed", "The speed of the "+fk+" temperature fan", nil, nil),
				prometheus.GaugeValue,
				fv.Speed)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_temperature", "The temperature of the "+fk+" temperature fan", nil, nil),
				prometheus.GaugeValue,
				fv.Temperature)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_temperature_sensor_"+metricName+"_target", "The target temperature for the "+fk+" temperature fan", nil, nil),
				prometheus.GaugeValue,
				fv.Target)
		}

		// output_pin
		for k, v := range result.Result.Status.OutputPins {
			metricName := getValidMetricName(k)
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc("klipper_output_pin_"+metricName+"_value", "The value of the "+k+" output pin", nil, nil),
				prometheus.GaugeValue,
				v.Value)
		}
	}
}

package main

import (
	"fmt"
	"math"
)

var (
	Styles []ReportingStyle = []ReportingStyle{
		diameterReporting{},
		areaReporting{},
	}
)

type ReportingStyle interface {
	Apply(m Measurement) Measurement
	Type() string
}

// The reported size of the touch is relative to the diameter of the contact.
type diameterReporting struct{}

func (d diameterReporting) Apply(m Measurement) Measurement {
	return m
}

func (d diameterReporting) Type() string {
	return "diameter"
}

// The reported size of the touch is relative to the area of the contact.
type areaReporting struct{}

func (a areaReporting) Apply(m Measurement) Measurement {
	return Measurement{m.Physical, math.Sqrt(m.Reported)}
}

func (a areaReporting) Type() string {
	return "area"
}

type Measurement struct {
	// The physical size of the touch in mm
	Physical float64
	// The reported size of the touch as a unit-less metric. This is the number produced by the
	// kernel for ABS_MT_TOUCH_MAJOR.
	Reported float64
}

type OptimizationResult struct {
	Type        string
	Scale, Bias float64
	Error       float64
}

func (o OptimizationResult) String() string {
	return fmt.Sprintf("OptimizationResult{Type=%s, Scale=%f, Bias=%f, Error=%f}",
		o.Type, o.Scale, o.Bias, o.Error)
}

func main() {
	// Consume input to get a list of (reported size, physical size) pairs.
	measurements := getMeasurements()
	dpi := getDpi()
	// For each reporting style, do data fitting to find the best parameters
	scaledMeasurements := make([]Measurement, len(measurements))
	results := make([]OptimizationResult, len(Styles))
	for i, style := range Styles {
		for j, m := range measurements {
			scaledMeasurements[j] = style.Apply(m)
		}
		scale, bias := findScaleAndBias(scaledMeasurements)
		stdError := calculateError(scaledMeasurements, scale, bias)
		results[i] = OptimizationResult{style.Type(), scale, bias, stdError}
	}
	// Using the optimal parameters, calculate the error for each type of size data
	bestResult := results[0]
	for _, r := range results {
		if r.Error < bestResult.Error {
			bestResult = r
		}
	}
	fmt.Println(bestResult)
	// Produce an idc file with the appropriate parameters
	fmt.Printf("Bias=%f, Scale=%f\n", dpi*bestResult.Bias, dpi*bestResult.Scale)
}

func getMeasurements() []Measurement {
	return []Measurement{
		Measurement{4.85, 6},
		Measurement{6.9, 8},
		Measurement{8.85, 11},
		Measurement{11, 14},
		Measurement{13.91, 18},
		Measurement{21.91, 28},
	}
}

func getDpi() float64 {
	return 16.61
}

func findScaleAndBias(ms []Measurement) (float64, float64) {
	// Optimize y = alpha + beta * x, where x = Reported, y = Physical
	temp := make([]float64, len(ms))

	for i := range ms {
		temp[i] = ms[i].Reported
	}
	avgReport := average(temp)
	stdDevReport := stddev(temp, avgReport)

	// Get the average of the reported values squared
	for i := range ms {
		temp[i] = ms[i].Reported * ms[i].Reported
	}
	avgReportSquared := average(temp)

	for i := range ms {
		temp[i] = ms[i].Physical
	}
	avgPhysical := average(temp)
	stdDevPhysical := stddev(temp, avgPhysical)

	for i := range ms {
		temp[i] = ms[i].Physical * ms[i].Physical
	}
	avgPhysicalSquared := average(temp)

	for i := range ms {
		temp[i] = ms[i].Reported * ms[i].Physical
	}
	avgReportPhysical := average(temp)

	corrCoeffNum := avgReportPhysical - avgReport*avgPhysical
	corrCoeffDenom := avgReportSquared - avgReport*avgReport
	corrCoeffDenom *= avgPhysicalSquared - avgPhysical*avgPhysical
	corrCoeffDenom = math.Sqrt(corrCoeffDenom)
	corrCoeff := corrCoeffNum / corrCoeffDenom

	beta := corrCoeff * (stdDevPhysical / stdDevReport)
	alpha := avgPhysical - beta*avgReport
	return beta, alpha
}

func calculateError(ms []Measurement, scale, bias float64) float64 {
	sum := float64(0)
	for _, m := range ms {
		estimate := m.Reported*scale + bias
		diff := m.Physical - estimate
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(ms)))
}

func average(nums []float64) float64 {
	sum := float64(0)
	for _, val := range nums {
		sum += val
	}
	return sum / float64(len(nums))
}

func stddev(nums []float64, avg float64) float64 {
	sum := float64(0)
	for _, val := range nums {
		dev := val - avg
		sum += dev * dev
	}
	return math.Sqrt(sum / float64(len(nums)))
}

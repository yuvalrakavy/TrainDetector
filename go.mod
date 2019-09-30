module github.com/yuvalrakavy/TrainDetector

go 1.13

replace github.com/yuvalrakavy/TrainDetectorCommon => ../TrainDetectorCommon

require (
	github.com/stianeikeland/go-rpio v4.2.0+incompatible // indirect
	github.com/stianeikeland/go-rpio/v4 v4.4.0
	github.com/yuvalrakavy/TrainDetectorCommon v0.0.0-20190922191303-cee2b3eae919
	github.com/yuvalrakavy/goPool v1.0.0
	github.com/yuvalrakavy/goRaspberryPi v1.0.1-0.20190923185026-bbf21c911540
	golang.org/x/sys v0.0.0-20190922100055-0a153f010e69 // indirect
)

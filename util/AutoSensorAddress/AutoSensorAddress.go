package main

import (
	"fmt"

	"github.com/stianeikeland/go-rpio/v4"
	"github.com/yuvalrakavy/goRaspberryPi/i2c"
	"github.com/yuvalrakavy/goRaspberryPi/vl6180x"
)

const resetVl6180xPinNumber = 4

/*
func ProgramNextAddress(bus *i2c.I2Cbus, nextAddress byte) error {
	if err := vl6180x.IsVL6180x(bus, byte(41)); err != nil {
		return err // No sensor was taken out of reser
	}

	sensor := vl6180x.Device(bus, 41)

	sensor.SetGPIO1low() // Put next sensor in reset state
	if err := sensor.Initialize(); err != nil {
		return err
	}

	if err := sensor.SetAddress(nextAddress); err != nil {
		return err
	}

	sensor.SetGPIO1high() // Take next sensor out of reset

	// Sleep 400ms to allow next sensor to recover
	time.Sleep(400 * time.Nanosecond * 1000000)

	return nil
}
*/

func main() {
	err := rpio.Open()
	if err != nil {
		panic(err)
	}
	defer rpio.Close()

	i2cBus, err := i2c.Open(1)
	if err != nil {
		panic(err)
	}
	defer i2cBus.Close()

	resetVl6180xPin := rpio.Pin(resetVl6180xPinNumber)
	resetVl6180xPin.Output()
	resetVl6180xPin.PullUp()

	sensors, err := vl6180x.AssignAddresses(i2cBus, 20, func() { resetVl6180xPin.Low() }, func() { resetVl6180xPin.High() })
	if err != nil {
		panic(err)
	}

	fmt.Println("Found ", len(sensors), "sensors:")
	for _, sensor := range sensors {
		fmt.Println("  at address:", sensor.Address)
	}
}

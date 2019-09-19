package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/yuvalrakavy/goRaspberryPi/i2c"
	"github.com/yuvalrakavy/goRaspberryPi/vl6180x"
)

func showUsage() {
	fmt.Println("Usage:")
	fmt.Println("  SetSensorAddress                             -- Show vl6180x on bus")
	fmt.Println("  SetSensorAddress <new-address>               -- Set sensor address")
	fmt.Println("  SetSensorAddress <old-address> <new-address> -- Change sensor address from <old> to <new> address")
}

func main() {
	var err error

	if len(os.Args) > 1 && os.Args[1][0] == '-' {
		showUsage()
		os.Exit(1)
	}

	var bus *i2c.I2Cbus

	bus, err = i2c.Open(1)
	if err != nil {
		panic(err)
	}

	defer bus.Close()

	sensors, err := scanBus(bus)

	if err != nil {
		panic(err)
	}

	if len(sensors) == 0 {
		fmt.Println("No vl6180x sensors were found")
	} else {
		if len(os.Args) < 2 {
			var noun = "sensor"

			if len(sensors) > 1 {
				noun = "sensors"
			}

			fmt.Printf("Found %d vl6180x %s on the bus:\n", len(sensors), noun)

			for _, sensor := range sensors {
				fmt.Println("  at address: ", sensor.Address)
			}
		} else if len(os.Args) < 4 {
			if err = programAddress(os.Args, sensors); err != nil {
				fmt.Println(err)
			}
		} else {
			showUsage()
		}
	}
}

func programAddress(args []string, sensors []vl6180x.Vl6180x) error {
	var (
		newAddress      byte
		oldAddress      byte
		err             error
		newAddressIndex int
	)

	if len(args) == 3 { // Old and new address were provided
		if oldAddress, err = parseAddress(os.Args[1]); err != nil {
			return nil
		}

		if onBus(sensors, oldAddress) == nil {
			return fmt.Errorf("Bus has no sensor with address %d, wrong old sensor address was specified", oldAddress)
		}

		newAddressIndex = 2
	} else {
		if len(sensors) > 1 {
			return fmt.Errorf("Bus has more than one sensor, please explicitly specify the address you want to chnage (i.e. SetSensorAddress <old-address> %d", newAddress)
		} else {
			oldAddress = sensors[0].Address
		}

		newAddressIndex = 1
	}

	if newAddress, err = parseAddress(os.Args[newAddressIndex]); err != nil {
		return err
	}

	if onBus(sensors, newAddress) != nil {
		return fmt.Errorf("Sensor with address %d is already on the bus", newAddress)
	}

	sensorToProgram := onBus(sensors, oldAddress)

	if sensorToProgram == nil {
		panic("SensorToProgram is nil")
	}

	if err = sensorToProgram.SetAddress(newAddress); err != nil {
		return err
	}

	fmt.Println("Sensor with address", oldAddress, " was reprogrammed to address", newAddress)
	return nil
}

func scanBus(bus *i2c.I2Cbus) ([]vl6180x.Vl6180x, error) {
	sensors := make([]vl6180x.Vl6180x, 0, 10)

	for address := byte(0); address < 127; address++ {
		if vl6180x.IsVL6180x(bus, address) == nil {
			sensors = append(sensors, vl6180x.Device(bus, address))
		}
	}

	return sensors, nil
}

func parseAddress(s string) (byte, error) {
	if addressArgValue, err := strconv.ParseInt(s, 0, 8); err != nil {
		return 0, err
	} else {
		return byte(addressArgValue), nil
	}
}

func onBus(sensors []vl6180x.Vl6180x, address byte) *vl6180x.Vl6180x {
	for _, sensor := range sensors {
		if sensor.Address == address {
			return &sensor
		}
	}

	return nil
}

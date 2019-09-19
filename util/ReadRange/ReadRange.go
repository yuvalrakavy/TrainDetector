package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/yuvalrakavy/goRaspberryPi/i2c"
	"github.com/yuvalrakavy/goRaspberryPi/vl6180x"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: ReadRange <Sensor Address>")
		return
	}

	sensorAddress, err := strconv.ParseInt(os.Args[1], 0, 8)
	if err != nil {
		panic(err)
	}

	bus, err := i2c.Open(1)
	if err != nil {
		panic(err)
	}

	defer bus.Close()

	if err = vl6180x.IsVL6180x(bus, byte(sensorAddress)); err != nil {
		fmt.Println("Address", sensorAddress, " has no valid Vl6180x sensor")
		return
	}

	sensor := vl6180x.Device(bus, byte(sensorAddress))

	if err = sensor.Initialize(); err != nil {
		panic(err)
	}

	done := make(chan interface{})

	go func(done <-chan interface{}) {
		for {
			select {
			case <-done:
				return

			default:
				if distance, err := sensor.ReadRange(0); err != nil {
					panic(err)
				} else {
					fmt.Println("  Sensor", sensor.Address, " range ", distance)
				}
			}
		}
	}(done)

	reader := bufio.NewReader(os.Stdin)
	if _, err = reader.ReadString('\n'); err != nil {
		panic(err)
	}

	close(done)
}

package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/yuvalrakavy/goPool"
	"github.com/yuvalrakavy/goRaspberryPi/i2c"
	"github.com/yuvalrakavy/goRaspberryPi/vl6180x"
)

type IsSensorCoveredMessage struct {
	Sensor    vl6180x.Vl6180x
	IsCovered bool
}

func endPipe(pool *goPool.GoPool, ch <-chan IsSensorCoveredMessage) {
	go func() {
		pool.Enter()
		defer pool.Leave()

		for {
			select {
			case <-pool.Done:
				fmt.Println("PIPE CLOSED")
				return
			case <-ch:
			}
		}
	}()
}

func dumpRangeMessages(pool *goPool.GoPool, rangeMessageChannel <-chan vl6180x.RangeValueMessage) (*goPool.GoPool, <-chan vl6180x.RangeValueMessage) {
	outputChannel := make(chan vl6180x.RangeValueMessage, 5)

	go func() {
		pool.Enter()
		defer pool.Leave()
		defer close(outputChannel)

		for {
			select {
			case <-pool.Done:
				fmt.Println("dumpRangeMessages terminated")
				return

			case rangeMessage := <-rangeMessageChannel:
				fmt.Println("Sensor ", rangeMessage.Sensor.Address, "distance:", rangeMessage.Distance)
				outputChannel <- rangeMessage
			}
		}
	}()

	return pool, outputChannel
}

func dumpIsCoveredMessage(pool *goPool.GoPool, isCoveredMessageChannel <-chan IsSensorCoveredMessage) (*goPool.GoPool, <-chan IsSensorCoveredMessage) {
	outputChannel := make(chan IsSensorCoveredMessage, 5)

	go func() {
		pool.Enter()
		defer pool.Leave()
		defer close(outputChannel)

		for {
			select {
			case <-pool.Done:
				fmt.Println("dumpIsCoveredMessage terminated")
				return

			case m := <-isCoveredMessageChannel:
				if m.IsCovered {
					fmt.Println("Sensor", m.Sensor.Address, " is covered")
				} else {
					fmt.Println("Sensor", m.Sensor.Address, " is not covered")
				}

				outputChannel <- m
			}
		}
	}()

	return pool, outputChannel
}

func getTransformRangeToIsCovered(pool *goPool.GoPool, rangeReadingChannels <-chan vl6180x.RangeValueMessage) (*goPool.GoPool, <-chan IsSensorCoveredMessage) {
	isCoveredChannel := make(chan IsSensorCoveredMessage)

	go func() {
		defer close(isCoveredChannel)
		pool.Enter()
		defer pool.Leave()
		currentValues := make(map[byte]bool)

		for {
			select {
			case <-pool.Done:
				fmt.Println("getTransformRangeToIsCovered terminated")
				return

			case rangeValueMessage := <-rangeReadingChannels:
				var isCovered bool

				if rangeValueMessage.Distance != 0xff {
					isCovered = true
				}

				currentlyCovered, hasCurrentValue := currentValues[rangeValueMessage.Sensor.Address]

				if !hasCurrentValue || currentlyCovered != isCovered {
					currentValues[rangeValueMessage.Sensor.Address] = isCovered
					isCoveredChannel <- IsSensorCoveredMessage{Sensor: rangeValueMessage.Sensor, IsCovered: isCovered}
				}
			}
		}
	}()

	return pool, isCoveredChannel
}

func main() {
	fmt.Println("vl6180x Testing")

	bus, err := i2c.Open(1)

	if err != nil {
		panic(err)
	}

	defer bus.Close()

	sensors, err := vl6180x.ScanBus(bus)

	if err != nil {
		panic(err)
	}

	if err = sensors.Initialize(); err != nil {
		panic(err)
	}

	fmt.Println(len(sensors), "sensors have been initialized")

	pool := goPool.Make()
	//endPipe(dumpRangeMessages(sensors.GetRangeReadingChannel(p)))
	endPipe(dumpIsCoveredMessage(getTransformRangeToIsCovered(sensors.GetRangeReadingChannel(pool))))
	StartNetworkHandler(pool)

	reader := bufio.NewReader(os.Stdin)
	if _, err = reader.ReadString('\n'); err != nil {
		panic(err)
	}

	fmt.Println("Terminating pool")
	pool.Terminate()
	fmt.Println("Exiting")
}

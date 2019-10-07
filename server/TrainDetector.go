package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/stianeikeland/go-rpio/v4"

	"github.com/yuvalrakavy/TrainDetectorCommon/networkHandler"
	"github.com/yuvalrakavy/TrainDetectorCommon/networkProtocol"
	"github.com/yuvalrakavy/goPool"
	"github.com/yuvalrakavy/goRaspberryPi/i2c"
	"github.com/yuvalrakavy/goRaspberryPi/vl6180x"
)

const firstSensorAddress = 20

type SensorsDataHandlerMessage interface {
	sensorDataHandlerMessage()
}
type IsSensorCoveredMessage struct {
	Sensor    vl6180x.Vl6180x
	IsCovered bool
}

func (_ IsSensorCoveredMessage) sensorDataHandlerMessage() {}

type SensorsStateSnapshot struct {
	version uint32
	states  []bool
}

type GetSensorsStateMessage struct {
	replyChannel chan SensorsStateSnapshot
}

func (_ GetSensorsStateMessage) sensorDataHandlerMessage() {}

type SubscriptionHandlerMessage interface {
	subscriptionHandlerMessage()
}

type AddSubscriberMessage struct {
	address net.Addr
}

func (_ AddSubscriberMessage) subscriptionHandlerMessage() {}

type RemoveSubscriberMessage struct {
	address net.Addr
}

func (_ RemoveSubscriberMessage) subscriptionHandlerMessage() {}

type SendStateChangeNotificationMessage struct {
	changedSensorNumber int
	isCovered           bool
	snapshot            SensorsStateSnapshot
}

func (_ SendStateChangeNotificationMessage) subscriptionHandlerMessage() {}

type ServerConfig struct {
	name               string
	firstSensorAddress byte
	sensorsCount       int

	connection                 *networkHandler.Connection
	sensorDataHandlerChannel   chan SensorsDataHandlerMessage
	subscriptionHandlerChannel chan SubscriptionHandlerMessage
}

func getTransformRangeToIsCovered(pool *goPool.GoPool, rangeReadingChannels <-chan vl6180x.RangeValueMessage) <-chan IsSensorCoveredMessage {
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

	return isCoveredChannel
}

func main() {
	fmt.Println("vl6180x Testing")

	bus, err := i2c.Open(1)
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	err = rpio.Open()
	if err != nil {
		panic(err)
	}
	defer rpio.Close()

	log.Println("Initializing VL6180x sensors...")

	const resetVl6180xPinNumber = 4

	resetVl6180xPin := rpio.Pin(resetVl6180xPinNumber)
	resetVl6180xPin.Output()
	resetVl6180xPin.PullUp()

	sensors, err := vl6180x.AssignAddresses(bus, firstSensorAddress, func() { resetVl6180xPin.Low() }, func() { resetVl6180xPin.High() })
	if err != nil {
		panic(err)
	}

	sensorsCount := int(len(sensors))

	log.Println("Found ", sensorsCount, "sensors.")

	pool := goPool.Make()

	connection, err := networkHandler.StartIncomingPacketsHandling(pool, networkProtocol.TrainDetectorAddress, func(rawPacket networkHandler.RawPacket) networkHandler.PacketReplyParser {
		return networkHandler.PacketReplyParser(networkProtocol.TrainDetectorPacket(rawPacket.PacketBytes))
	})

	if err != nil {
		panic(err)
	}

	serverConfig := ServerConfig{
		name:               "Raspberry",
		firstSensorAddress: firstSensorAddress,
		sensorsCount:       sensorsCount,

		sensorDataHandlerChannel:   nil,
		subscriptionHandlerChannel: nil,
		connection:                 connection,
	}

	serverConfig.sensorDataHandlerChannel = StartSensorsDataHandler(pool, &serverConfig)
	serverConfig.subscriptionHandlerChannel = StartSubscriptionsHandler(pool, &serverConfig)

	go handleSensorsReading(pool, &serverConfig, sensors)
	go handleIncomingRawPackets(pool, &serverConfig)

	reader := bufio.NewReader(os.Stdin)
	if _, err = reader.ReadString('\n'); err != nil {
		panic(err)
	}

	fmt.Println("Terminating pool")
	pool.Terminate()
	fmt.Println("Exiting")
}

func StartSensorsDataHandler(pool *goPool.GoPool, serverConfig *ServerConfig) chan SensorsDataHandlerMessage {
	incomingMessagesChannel := make(chan SensorsDataHandlerMessage)
	sensorsData := make([]bool, serverConfig.sensorsCount)
	var version uint32 = 0

	getSensorsStateSnapshot := func() SensorsStateSnapshot {
		return SensorsStateSnapshot{
			// Clone the sensor states
			states:  append(sensorsData[:0:0], sensorsData...),
			version: version,
		}
	}

	go func() {
		pool.Enter()
		defer pool.Leave()

		for {
			select {
			case <-pool.Done:
				return

			case aMessage := <-incomingMessagesChannel:
				switch message := aMessage.(type) {
				case IsSensorCoveredMessage:
					index := int(message.Sensor.Address) - firstSensorAddress
					if index < 0 || index >= serverConfig.sensorsCount {
						log.Fatal("Data from invalid sensor address", message.Sensor)
					} else {
						if message.IsCovered {
							log.Println("Sensor", index, "is covered")
						} else {
							log.Println("Sensor", index, "is not covered")
						}

						version = version + 1
						sensorsData[index] = message.IsCovered

						serverConfig.subscriptionHandlerChannel <- SendStateChangeNotificationMessage{
							snapshot:            getSensorsStateSnapshot(),
							changedSensorNumber: index,
							isCovered:           message.IsCovered,
						}
					}

				case GetSensorsStateMessage:
					message.replyChannel <- getSensorsStateSnapshot()

				default:
					log.Fatal("Unexpected message to sensorsDataHandler", message)
				}
			}
		}
	}()

	return incomingMessagesChannel
}

func StartSubscriptionsHandler(pool *goPool.GoPool, serverConfig *ServerConfig) chan SubscriptionHandlerMessage {
	incomingMessagesChannel := make(chan SubscriptionHandlerMessage)
	subscriptions := make(map[string]net.Addr)

	go func() {
		pool.Enter()
		defer pool.Leave()

		for {
			select {
			case <-pool.Done:
				return

			case aMessage := <-incomingMessagesChannel:
				switch message := aMessage.(type) {
				case AddSubscriberMessage:
					log.Println(message.address.String(), "subscribed to state change notifications")
					subscriptions[message.address.String()] = message.address

				case RemoveSubscriberMessage:
					log.Println(message.address.String(), "unsubscribed from state change notifications")
					delete(subscriptions, message.address.String())

				case SendStateChangeNotificationMessage:
					for _, subscriberAddress := range subscriptions {
						go func(address net.Addr) {
							_, ok := <-serverConfig.connection.CreateRequest(
								func(requestNumber int) {
									packetBytes := networkProtocol.NewStateChangedNotificationPacket(
										requestNumber,
										byte(message.changedSensorNumber),
										message.isCovered,
										message.snapshot.version,
										message.snapshot.states,
									).Encode()

									if _, err := serverConfig.connection.UdpConnection.WriteTo(packetBytes, address); err != nil {
										log.Println("Error sending notification to ", address, ":", err)
									}
								},
								300*time.Millisecond, 3,
							)

							if !ok {
								log.Println("Did not receive reply on state changed notification from", address.String(), " removing its subscription")
								// If no reply was received from subscriber (after retries), remove this subscriber
								serverConfig.subscriptionHandlerChannel <- RemoveSubscriberMessage{address: address}
							}
						}(subscriberAddress)
					}
				}
			}
		}

	}()

	return incomingMessagesChannel
}

func handleSensorsReading(pool *goPool.GoPool, serverConfig *ServerConfig, sensors vl6180x.Vl6180xGroup) {
	pool.Enter()
	defer pool.Leave()

	isCoveredMessageChannel := getTransformRangeToIsCovered(sensors.GetRangeReadingChannel(pool))

	for {
		select {
		case <-pool.Done:
			return

		case message := <-isCoveredMessageChannel:
			serverConfig.sensorDataHandlerChannel <- message
		}
	}
}

func handleIncomingRawPackets(pool *goPool.GoPool, serverConfig *ServerConfig) {
	pool.Enter()
	defer pool.Leave()

	for {
		select {
		case <-pool.Done:
			return

		case rawPacket := <-serverConfig.connection.IncomingRequestsChannel:
			err := handlePacket(serverConfig, rawPacket)
			if err != nil {
				log.Println("Incoming packet decoding returned error:", err.Error())
			}
		}
	}
}

func handlePacket(serverConfig *ServerConfig, rawPacket networkHandler.RawPacket) error {
	packetBytes := networkProtocol.TrainDetectorPacket(rawPacket.PacketBytes)
	packetReader := packetBytes.Reader()

	aPacket, err := packetReader.Decode()
	if err != nil {
		return err
	}

	switch inputPacket := aPacket.(type) {

	case *networkProtocol.PleaseIdentifyPacket:

		go func(replyChannel chan *networkHandler.RawPacket) {
			<-replyChannel // Nothing to do with the reply
		}(serverConfig.connection.CreateRequest(func(requestNumber int) {
			identificationInfoPacket := networkProtocol.NewIdentificationInfo(
				requestNumber,
				int(serverConfig.sensorsCount),
				networkProtocol.PacketString(serverConfig.name),
			)

			identificationInfoBytes := identificationInfoPacket.Encode()
			if _, err := serverConfig.connection.UdpConnection.WriteTo(identificationInfoBytes, rawPacket.RemoteAddress); err != nil {
				log.Fatal(err)
			}
		}, 300*time.Millisecond, 3))

	case *networkProtocol.GetStateRequestPacket:
		statesReplyChannel := make(chan SensorsStateSnapshot)
		serverConfig.sensorDataHandlerChannel <- GetSensorsStateMessage{replyChannel: statesReplyChannel}

		snapshot := <-statesReplyChannel
		if err := rawPacket.SendReply(serverConfig.connection.UdpConnection, networkProtocol.NewGetStateReply(inputPacket.Header.RequestNumber(), snapshot.version, snapshot.states)); err != nil {
			log.Fatal(err)
		}

	case *networkProtocol.SubscribeRequestPacket:
		serverConfig.subscriptionHandlerChannel <- AddSubscriberMessage{address: rawPacket.RemoteAddress}

		if err := rawPacket.SendReply(serverConfig.connection.UdpConnection, networkProtocol.NewSubscribeReplyPacket(inputPacket.Header.RequestNumber())); err != nil {
			log.Fatal(err)
		}

	case *networkProtocol.UnsubscribeRequestPacket:
		serverConfig.subscriptionHandlerChannel <- RemoveSubscriberMessage{address: rawPacket.RemoteAddress}

		if err := rawPacket.SendReply(serverConfig.connection.UdpConnection, networkProtocol.NewUnsubscribeReplyPacket(inputPacket.Header.RequestNumber())); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

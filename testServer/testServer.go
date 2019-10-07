package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yuvalrakavy/TrainDetectorCommon/networkHandler"
	"github.com/yuvalrakavy/TrainDetectorCommon/networkProtocol"
	"github.com/yuvalrakavy/goPool"
)

type Controller struct {
	Address         net.Addr
	Name            string
	ProtocolVersion byte
	SensorsCount    int
}

type Controllers []Controller

type ConnectionInfo struct {
	connection                 *networkHandler.Connection
	registerControllersChannel chan interface{}
}

type ControllerRegisterAddMessage struct {
	indentificationInfoPacket *networkProtocol.IndenticiationInfoPacket
	remoteAddress             net.Addr
}

type ControllerGetListMessage struct {
	replyChannel chan Controllers
}

type StateSnapshot struct {
	version uint32
	states  []bool
}

func main() {

	fmt.Println("Train Detector server tester")

	pool := goPool.Make()

	// Start processing incoming UDP packets
	connection, err := networkHandler.StartIncomingPacketsHandling(pool, ":0", func(rawPacket networkHandler.RawPacket) networkHandler.PacketReplyParser {
		return networkHandler.PacketReplyParser(networkProtocol.TrainDetectorPacket(rawPacket.PacketBytes))
	})

	connectionInfo := ConnectionInfo{
		connection:                 connection,
		registerControllersChannel: make(chan interface{}),
	}

	if err != nil {
		panic(err)
	}

	go handleIncomingRawPackets(pool, &connectionInfo)
	go registerControllers(pool, connectionInfo)

	commandLoop(&connectionInfo)

	pool.Terminate()
}

func handleIncomingRawPackets(pool *goPool.GoPool, connectionInfo *ConnectionInfo) {
	pool.Enter()
	defer pool.Leave()

	for {
		select {
		case <-pool.Done:
			return

		case rawPacket := <-connectionInfo.connection.IncomingRequestsChannel:
			packetBytes := networkProtocol.TrainDetectorPacket(rawPacket.PacketBytes)
			packetReader := packetBytes.Reader()

			aPacket, err := packetReader.Decode()
			if err != nil {
				log.Println("Incoming packet decoding returned error:", err.Error())
			} else {
				fmt.Printf("Received packet %T from %v\n", aPacket, rawPacket.RemoteAddress)

				switch packet := aPacket.(type) {

				case *networkProtocol.IndenticiationInfoPacket:
					connectionInfo.registerControllersChannel <- ControllerRegisterAddMessage{
						indentificationInfoPacket: packet,
						remoteAddress:             rawPacket.RemoteAddress,
					}

					if err := rawPacket.SendReply(connectionInfo.connection.UdpConnection, networkProtocol.NewIdentificationAcknowledge(packet.Header.RequestNumber())); err != nil {
						log.Println("Error sending identification Ack:", err)
					}

				case *networkProtocol.StateChangedNotificationPacket:
					fmt.Print("NOTIFICATION version: ", packet.Version, " - Sensor ", packet.SensorNumber)
					if packet.IsCovered {
						fmt.Println(" is covered")
					} else {
						fmt.Println(" is not covered")
					}

					if err := rawPacket.SendReply(connectionInfo.connection.UdpConnection, networkProtocol.NewStateChangedAcknowledgePacket(packet.Header.RequestNumber())); err != nil {
						log.Println("Error sending identification Ack:", err)
					}
				}
			}
		}
	}
}

func registerControllers(pool *goPool.GoPool, connectionInfo ConnectionInfo) {
	pool.Enter()
	defer pool.Leave()

	controllersMap := make(map[string]Controller)

	for {
		select {
		case <-pool.Done:
			return

		case aMessage := <-connectionInfo.registerControllersChannel:
			switch message := aMessage.(type) {
			case ControllerRegisterAddMessage:
				controllersMap[string(message.indentificationInfoPacket.Name)] = Controller{
					Name:            string(message.indentificationInfoPacket.Name),
					ProtocolVersion: message.indentificationInfoPacket.ProtocolVersion,
					Address:         message.remoteAddress,
					SensorsCount:    int(message.indentificationInfoPacket.SensorsCount),
				}

			case ControllerGetListMessage:
				controllers := make([]Controller, 0, len(controllersMap))

				for _, controller := range controllersMap {
					controllers = append(controllers, controller)
				}

				message.replyChannel <- controllers
			}
		}
	}
}

func commandLoop(connectionInfo *ConnectionInfo) {
	var currentController *Controller = nil
	var controllers Controllers = nil
	reader := bufio.NewReader(os.Stdin)

	hasController := func() bool {
		if currentController == nil {
			fmt.Println("Please use the 'i' command to select controller")
		}

		return currentController != nil
	}

	for {
		if currentController == nil {
			fmt.Print("> ")
		} else {
			fmt.Printf("[%s (%s)]> ", currentController.Name, currentController.Address.String())
		}

		command, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		fields := strings.Fields(command)

		if len(fields) > 0 {
			switch fields[0] {
			case "i":
				controllers = identifyCommand(connectionInfo)
				controllers.Show()

				if len(controllers) == 1 {
					currentController = &controllers[0]
				} else if len(controllers) > 1 {

					for {
						fmt.Printf("Enter controller number to use [1-%d]: ", len(controllers))
						controllerIndexString, err := reader.ReadString('\n')

						if err != nil {
							fmt.Println(err)
							continue
						}

						index, err := strconv.Atoi(controllerIndexString)

						if err == nil && (index < 1 || index > len(controllers)) {
							err = fmt.Errorf("Value sould be between 1 and %d", len(controllers))
						}

						if err != nil {
							fmt.Println(err)
						} else {
							currentController = &controllers[index-1]
							break
						}
					}
				}

			case "g":
				if hasController() {
					snapshot, err := getStateCommand(connectionInfo, currentController)

					if err != nil {
						fmt.Println(err)
					} else {
						snapshot.Show()
					}
				}

			case "s":
				if hasController() {
					err := subscribeCommand(connectionInfo, currentController)

					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Subscribed to state changed notifications")
					}
				}

			case "u":
				if hasController() {
					err := unsubscribeCommand(connectionInfo, currentController)

					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Unsubscribed from state changed notifications")
					}

				}

			case "q":
				return
			}
		}
	}
}

func (controllers Controllers) Show() {
	fmt.Println("     Name                   Address           Version   #Sensors")

	for index, controller := range controllers {
		fmt.Printf(
			"%-2d)  %-23s%-18v%-10v%-11v\n",
			index+1,
			controller.Name,
			controller.Address,
			controller.ProtocolVersion,
			controller.SensorsCount,
		)
	}
}

func (snapshot *StateSnapshot) Show() {
	fmt.Println(len(snapshot.states), "sensors, state version", snapshot.version)
	for sensorNumber, sensorState := range snapshot.states {
		fmt.Print(sensorNumber)
		if sensorState {
			fmt.Println(" is covered")
		} else {
			fmt.Println(" is not covered")
		}
	}
}

func identifyCommand(connectionInfo *ConnectionInfo) Controllers {
	pleaseIdentifyPacket := networkProtocol.NewPleaseIdentify()
	pleaseIdentifyBytes := pleaseIdentifyPacket.Encode()

	broadcastAddress, err := net.ResolveUDPAddr("udp", "255.255.255.255"+networkProtocol.TrainDetectorAddress)
	if err != nil {
		panic(err)
	}
	_, err = connectionInfo.connection.UdpConnection.WriteTo(pleaseIdentifyBytes, broadcastAddress)
	if err != nil {
		panic(err)
	}

	time.Sleep(1 * time.Second) // Allow for 1 seconds to gather replies

	getControllersChannel := make(chan Controllers)
	connectionInfo.registerControllersChannel <- ControllerGetListMessage{replyChannel: getControllersChannel}

	controllers := <-getControllersChannel

	return controllers
}

func getStateCommand(connectionInfo *ConnectionInfo, controller *Controller) (*StateSnapshot, error) {
	replyChannel := connectionInfo.connection.CreateRequest(
		func(requestNumber int) {
			packetBytes := networkProtocol.NewGetStateRequest(requestNumber).Encode()
			if _, err := connectionInfo.connection.UdpConnection.WriteTo(packetBytes, controller.Address); err != nil {
				fmt.Println(err)
			}
		}, 300*time.Millisecond, 3,
	)

	rawReplyPacket, gotReply := <-replyChannel

	if gotReply {
		packetReader := networkProtocol.TrainDetectorPacket(rawReplyPacket.PacketBytes).Reader()
		replyPacket, err := packetReader.Decode()

		if err != nil {
			return nil, err
		}

		getStateReplyPacket, isGetStateReply := replyPacket.(*networkProtocol.GetStateReplyPacket)
		if !isGetStateReply {
			return nil, fmt.Errorf("Reply packet is not GetStateReply")
		}

		return &StateSnapshot{
			version: getStateReplyPacket.Version,
			states:  getStateReplyPacket.States,
		}, nil
	} else {
		return nil, fmt.Errorf("No reply was received for GetState packet")
	}
}

func subscribeCommand(connectionInfo *ConnectionInfo, controller *Controller) error {
	replyChannel := connectionInfo.connection.CreateRequest(
		func(requestNumber int) {
			packetBytes := networkProtocol.NewSubcribeRequestPacket(requestNumber).Encode()

			_, err := connectionInfo.connection.UdpConnection.WriteTo(packetBytes, controller.Address)
			if err != nil {
				fmt.Println(err)
			}
		}, 300*time.Millisecond, 3,
	)

	_, gotReply := <-replyChannel

	if !gotReply {
		return fmt.Errorf("Did not get reply to subscribe request")
	}
	return nil
}

func unsubscribeCommand(connectionInfo *ConnectionInfo, controller *Controller) error {
	replyChannel := connectionInfo.connection.CreateRequest(
		func(requestNumber int) {
			packetBytes := networkProtocol.NewUnsubscribeRequestPacket(requestNumber).Encode()

			_, err := connectionInfo.connection.UdpConnection.WriteTo(packetBytes, controller.Address)
			if err != nil {
				fmt.Println(err)
			}
		}, 300*time.Millisecond, 3,
	)

	_, gotReply := <-replyChannel

	if !gotReply {
		return fmt.Errorf("Did not get reply to unsubscribe request")
	}
	return nil
}

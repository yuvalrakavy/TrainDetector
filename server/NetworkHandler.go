package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/yuvalrakavy/TrainDetector/common/trainDetectorProtocol"
	"github.com/yuvalrakavy/goPool"
)

const trainDetectorAddress = ":3000"
const maxPacketSize = 4096

func StartNetworkHandler(pool *goPool.GoPool) {

	go func() {
		pool.Enter()
		defer pool.Leave()

		if myAddress, err := net.ResolveUDPAddr("udp", trainDetectorAddress); err != nil {
			panic(err)
		} else {
			conn, err := net.ListenUDP("udp", myAddress)
			if err != nil {
				panic(err)
			}

			shouldTerminate := false

			go processPackets(conn, &shouldTerminate)

			<-pool.Done

			shouldTerminate = true
			fmt.Println("Closing UDP connection")
			conn.Close()
		}
	}()

}

func processPackets(conn *net.UDPConn, shouldTerminate *bool) {
	packetBuffer := make([]byte, maxPacketSize)

	for {
		packetLength, err := conn.Read(packetBuffer)

		if err != nil {
			if *shouldTerminate {
				fmt.Println("Packet processing terminated")
				return
			} else {
				panic(err)
			}
		}

		packetReader := bytes.NewReader(packetBuffer)

		var header trainDetectorProtocol.PacketHeader

		if err := binary.Read(packetReader, binary.LittleEndian, &header); err != nil {
			log.Println("Error reading packet header", err)
		}

		packetTypeDescription, validPacketDescription := trainDetectorProtocol.PacketTypeNameMap[header.PacketType]

		if validPacketDescription {
			log.Println("Received packet:", packetTypeDescription)
		}

		switch {
		case header.PacketType == trainDetectorProtocol.PacketTypeIdentifyRequest:
			err = handleIdetifyRequest(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeConfigRequest:
			err = handleConfigRequest(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeIdentifyAcknowledge:
			err = handleIdentifyAcknowledge(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeGetStateRequest:
			err = handleGetState(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeSubscribeRequst:
			err = handleSubscribe(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeUnsubscribeRequst:
			err = handleUnsubscribe(conn, packetReader, header)

		case header.PacketType == trainDetectorProtocol.PacketTypeStateChangedAcknowledge:
			err = handleStateChangedAcknowledge(conn, packetReader, header)

		default:
			err = fmt.Errorf("Received packet with invalid packet type %x (length %d bytes)", header.PacketType, packetLength)
		}

		if err != nil {
			log.Println("Error handling packet: ", err.Error())
		}
	}
}

func handleIdetifyRequest(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleConfigRequest(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleIdentifyAcknowledge(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleGetState(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleSubscribe(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleUnsubscribe(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

func handleStateChangedAcknowledge(conn *net.UDPConn, packetReader *bytes.Reader, header trainDetectorProtocol.PacketHeader) error {
	return nil
}

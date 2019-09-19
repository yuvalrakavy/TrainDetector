package trainDetectorProtocol

const TrainDetectorAddress = ":3000"

// VL6180 control registers
// nolint:go-lint,varcheck,deadcode
const (
	PacketClassRequest      = 0x00
	PacketClassReply        = 0x80
	PacketClassNotification = PacketClassRequest
	PacketClassAcknowledge  = PacketClassReply

	PacketTypeIdentify        = 0
	PacketTypeIdentifyRequest = PacketTypeIdentify | PacketClassRequest
	PacketTypeIdentifyReply   = PacketTypeIdentify | PacketClassReply

	PacketTypeIdentifyAcknowledge = 1 | PacketClassAcknowledge

	PacketTypeConfig        = 2
	PacketTypeConfigRequest = PacketTypeConfig | PacketClassRequest
	PacketTypeConfigReply   = PacketTypeConfig | PacketClassReply

	PacketTypeGetState        = 3
	PacketTypeGetStateRequest = PacketTypeGetState | PacketClassRequest
	PacketTypeGetStateReply   = PacketTypeGetState | PacketClassReply

	PacketTypeSubscribe       = 4
	PacketTypeSubscribeRequst = PacketTypeSubscribe | PacketClassRequest
	PacketTypeSubscribeReply  = PacketTypeSubscribe | PacketClassReply

	PacketTypeUnsubscribe       = 5
	PacketTypeUnsubscribeRequst = PacketTypeUnsubscribe | PacketClassRequest
	PacketTypeUnsubscribeReply  = PacketTypeUnsubscribe | PacketClassReply

	PacketTypeStateChanged             = 6
	PacketTypeStateChangedNotification = PacketTypeStateChanged | PacketClassNotification
	PacketTypeStateChangedAcknowledge  = PacketTypeStateChanged | PacketClassAcknowledge
)

var PacketTypeNameMap = map[byte]string{
	PacketTypeIdentifyRequest:          "Identify Request",
	PacketTypeIdentifyReply:            "Identify Reply",
	PacketTypeIdentifyAcknowledge:      "Identify reply acknowlede",
	PacketTypeConfigRequest:            "Config request",
	PacketTypeConfigReply:              "Config reply",
	PacketTypeGetStateRequest:          "GetState request",
	PacketTypeGetStateReply:            "GetState reply",
	PacketTypeSubscribeRequst:          "Subscribe request",
	PacketTypeSubscribeReply:           "Subscribe reply",
	PacketTypeUnsubscribeRequst:        "Unsubscribe request",
	PacketTypeUnsubscribeReply:         "Unsubscribe reply",
	PacketTypeStateChangedNotification: "State changed notification",
	PacketTypeStateChangedAcknowledge:  "State changed acknowledged",
}

type PacketHeader struct {
	PacketType byte
	Seq        byte
}

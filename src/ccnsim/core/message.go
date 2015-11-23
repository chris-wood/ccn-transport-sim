package core

type Message interface {
    GetName() string
    GetPayload() []uint8
    GetNonce() string

    ProcessAtRouter(router Router, face int)
    ProcessAtConsumer(consumer Consumer, face int)
    ProcessAtProducer(producer Producer, face int)
}

package ccnsim

import "fmt"

type Data struct {
    NetworkMessage
    Name string `json:"name"`
    Payload []uint8 `json:"payload"`
    Nonce string `json:"string"`
}

func (d Data) GetName() (string) {
    return d.Name;
}

func (d Data) GetPayload() ([]uint8) {
    return d.Payload;
}

func (d Data) ProcessAtRouter(router Router, arrivalFace int) {
    name := d.GetName();
    if router.Fwd.Pit.IsPending(name) {
        entries := router.Fwd.Pit.GetEntries(name);
        router.Fwd.Pit.RemoveEntry(name);

        router.Fwd.Cache.AddData(d);

        // forward to all entries...
        for i := 0; i < len(entries); i++ {
            entry := entries[i];
            targetFace := entry.arrivalFace;
            networkMsg := NetworkMessage{Nonce: entry.msg.GetNonce(), HopCount: entry.msg.GetHopCount() - 1};
            data := Data_CreateSimple(networkMsg, d.GetName(), d.GetPayload(), entry.msg.GetNonce());
            router.SendData(data, arrivalFace, targetFace); // strategy: first record in the longest FIB entry
        }
    } else {
        fmt.Println("not pending!!");
    }
}

func (d Data) ProcessAtConsumer(consumer Consumer, face int) {
    fmt.Printf("consumer processing data %s %s %d\n", d.GetName(), d.GetNonce(), d.GetHopCount());
}

func (d Data) ProcessAtProducer(producer Producer, face int) {
    fmt.Println("producer processing data -- WTF");
}

func (d Data) GetNonce() (string) {
    return d.Nonce;
}

func Data_CreateSimple(netMsg NetworkMessage, name string, payload []uint8, nonce string) (Data) {
    Data := Data{NetworkMessage: netMsg, Name: name, Payload: payload, Nonce: nonce};
    return Data;
}

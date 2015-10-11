package ccnsim

import "fmt"

type Interest struct {
    NetworkMessage
    Name string `json:"name"`
    Payload []uint8 `json:"payload"`
    HashRestriction []uint8 `json:"hashRestriction"`
    KeyIdRestriction []uint8 `json:"keyIdRestriction"`
}

func (i Interest) GetName() (string) {
    return i.Name;
}

func (i Interest) GetPayload() ([]uint8) {
    return i.Payload;
}

func (i Interest) GetNonce() (string) {
    return i.Nonce;
}

func (i Interest) ProcessAtRouter(router Router, arrivalFace int) {
    name := i.GetName();

    isInCache := router.Fwd.Cache.HasData(i.GetName())
    if isInCache {
        data := router.Fwd.Cache.GetData(i.GetName());

        networkMsg := NetworkMessage{Nonce: i.GetNonce(), HopCount: i.GetHopCount()};
        newData := Data_CreateSimple(networkMsg, data.GetName(), data.GetPayload(), i.GetNonce());
        router.SendData(newData, arrivalFace, arrivalFace);
    } else if router.Fwd.Pit.IsPending(name) {
        router.Fwd.Pit.AddEntry(i, arrivalFace);
    } else {

        // Insert into the PIT
        router.Fwd.Pit.AddEntry(i, arrivalFace);

        // Forward along
        faces, err := router.Fwd.Fib.GetInterfacesForPrefix(i.GetName());
        if err == nil {
            targetFace := faces[0]; // strategy: first record in the longest FIB entry
            networkMsg := NetworkMessage{i.GetNonce(), i.GetHopCount() - 1};
            newInterest := i.Copy(networkMsg);
            router.SendInterest(newInterest, arrivalFace, targetFace);
        } else {
            fmt.Println(err.Error());
            // NOT IN FIB, drop.
        }
    }
}

func (i Interest) ProcessAtConsumer(consumer Consumer, arrivalFace int) {
    fmt.Println("****consumer processing INTEREST");
}

func (i Interest) ProcessAtProducer(producer Producer, arrivalFace int) {
    networkMsg := NetworkMessage{Nonce: i.GetNonce(), HopCount: i.GetHopCount()};
    data := Data_CreateSimple(networkMsg, i.GetName(), i.GetPayload(), i.GetNonce());
    producer.SendData(data, arrivalFace);
}

func (i Interest) Copy(networkMsg NetworkMessage) (Interest) {
    Interest := Interest{NetworkMessage: networkMsg, Name: i.GetName(), Payload: i.GetPayload(),
        HashRestriction: i.HashRestriction, KeyIdRestriction: i.KeyIdRestriction};
    return Interest;
}

func Interest_CreateSimple(name string) (Interest) {
    nonce := makeNonce(10);
    networkMsg := NetworkMessage{ Nonce: nonce, HopCount: 100 };
    Interest := Interest{NetworkMessage: networkMsg, Name: name, Payload: []uint8{0x00},
        HashRestriction: []uint8{0x00}, KeyIdRestriction: []uint8{0x00}};
    return Interest;
}

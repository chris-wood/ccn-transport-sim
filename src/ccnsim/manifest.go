package ccnsim

import "fmt"

type Manifest struct {
    Name string `json:"name"`
    Contents []Data `json:"contents"`
}

func (m Manifest) ProcessAtRouter(router Router, face int) {
    fmt.Println("router processing manifest");
}

func (m Manifest) ProcessAtConsumer(consumer Consumer, face int) {
    fmt.Println("consumer processing manifest");
}

func (m Manifest) ProcessAtProducer(producer Producer, face int) {
    fmt.Println("producer processing manifest");
}

func Manifest_CreateSimple(datas []Data) (Manifest) {
    return Manifest{};
}

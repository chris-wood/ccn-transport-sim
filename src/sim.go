package main

import "fmt"
import "os"
import "strconv"
import "container/list"
import "encoding/json"

type Message interface {
    GetName() string
    GetPayload() []uint8
}

type Interest struct {
    Name string `json:"name"`
    Payload []uint8 `json:"payload"`
    HashRestriction []uint8 `json:"hashRestriction"`
    KeyIdRestriction []uint8 `json:"keyIdRestriction"`
}

func (i Interest) getName() (string) {
    return i.Name;
}

func (i Interest) getPayload() ([]uint8) {
    return i.Payload;
}

func Interest_CreateSimple(name string) (Interest) {
    Interest := Interest{Name: name, Payload: []uint8{0x00}, HashRestriction: []uint8{0x00}, KeyIdRestriction: []uint8{0x00}};
    return Interest;
}

type Data struct {
    Name string `json:"name"`
    Payload []uint8 `json:"payload"`
}

func (d Data) getName() (string) {
    return d.Name;
}

func (d Data) getPayload() ([]uint8) {
    return d.Payload;
}

func Data_CreateSimple(name string) (Data) {
    Data := Data{Name: name, Payload: []uint8{0x00}};
    return Data;
}

type Manifest struct {
    Name string `json:"name"`
    Contents []Data `json:"contents"`
}

func Manifest_CreateSimple(datas []Data) (Manifest) {
    // TODO: make this a real Manifest...
    return Manifest{};
}

type StagedMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
}

type fibentry struct {
    Prefix string `json:"prefix"`
    Interfaces []int `json:"interface"`
}

type fibtable struct {
    Table map[string]fibentry
}

type pitentry struct {
    Name string
    Records []Interest
}

type pittable struct {
    Table map[string]pitentry
}

type cache struct {
    Cache map[string]Data
}

type link struct {
    Stage list.List `json:"stage"`
    Pfail float32 `json:"pfail"`
    Rate float32 `json:"rate"`
}

func (l link) txTime(size int) (int) {
    pktTime := int(float32(size) / l.Rate);
    return pktTime;
}

type queue struct {
    Fifo list.List `json:"fifo"`
    Capacity int `json:"capacity"`
}

func (q queue) PushBackInterest(msg Interest) {
    q.Fifo.PushBack(msg);
    fmt.Print("Inserting: ");

    json, _ := json.Marshal(msg)
    fmt.Println(string(json));
}

func (q queue) PushBackData(msg Data) {
    q.Fifo.PushBack(msg);
}

func (q queue) PushBackManifest(msg Manifest) {
    q.Fifo.PushBack(msg);
}

type Forwarder struct {
    Identity string

    Faces []int
    FaceQueues map[int]*queue
    FaceLinks map[int]*link

    Fib *fibtable
    Cache *cache
    Pit *pittable
}

type Consumer struct {
    *Forwarder
}

type Runnable interface {
    Tick()
}

func (c Consumer) Tick(time int) {

    fmt.Printf("Tick = %d\n", time);

    // Handle links
    for i := 0; i < len(c.Faces); i++ {

        faceId := c.Faces[i];
        link := c.FaceLinks[faceId];

        lj, _ := json.Marshal(link.Stage);
        fmt.Println(string(lj));

        for e := link.Stage.Front(); e != nil; e = e.Next() {
            fmt.Printf("Moving message at %d\n", time);
            msg := e.Value.(StagedMessage);
            if (msg.TicksLeft > 0) {
                msg.TicksLeft--;
            } else {
                link.Stage.Remove(e);
                // TODO: drop it in the recipient queue here...
            }
        }
    }

    // Handle queue movement...
    for i := 0; i < len(c.Faces); i++ { // length == 1
        faceId := c.Faces[i];
        queue := c.FaceQueues[faceId];
        link := c.FaceLinks[faceId];
        fmt.Print("stage = ");
        fmt.Println(link.Stage);
        fmt.Print("queue = ");
        fmt.Println(queue.Fifo);
        fmt.Printf("queue size = %d\n", queue.Fifo.Len());

        for e := queue.Fifo.Front(); e != nil; e = e.Next() {
            // simulate processing delay
            link.Stage.PushBack(e.Value);

            fmt.Printf("link stage size = %d\n", link.Stage.Len());
            // queue.fifo.Remove(e);

            // TODO: this is not moving the message to the link staging area
        }
    }
}

func (c Consumer) SendInterest(msg Interest) {
    defaultFace := c.Faces[0];
    queue := c.FaceQueues[defaultFace];
    queue.PushBackInterest(msg);
}

func (c Consumer) ReceiveInterest(msg Interest) {
    fmt.Println("received Interest");
}

func (c Consumer) ReceiveData(msg Data) {
    fmt.Println("received Data");
}

func (c Consumer) ReceiveManifest(msg Manifest) {
    fmt.Println("received Manifest");
}

func consumer_Create(id string) (*Consumer) {
    faceMap := make(map[int]*queue);
    faceLinkMap := make(map[int]*link);
    fifo := &queue{*list.New(), 100000};
    link := &link{*list.New(), 0.0, 1000}

    defaultFace := 1;
    faceMap[defaultFace] = fifo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace}, faceMap, faceLinkMap, &fibtable{}, &cache{}, &pittable{}};
    consumer := &Consumer{fwd};
    return consumer;
}

type Producer struct {
    *Forwarder
}

func (p Producer) Tick(time int) {
    //
}

type Router struct {
    *Forwarder
}

func (r Router) Tick(time int) {
    //
}

type Event struct {
    desc string
    val int
}

func main() {
    fmt.Println("ccn-transport-sim v0.0.0");

    args := os.Args[1:]

    simulationTime, err := strconv.Atoi(args[0]);
    if err != nil {
        fmt.Printf("Error: invalid simulation time %s\n", args[0]);
        os.Exit(-1);
    }

    events := list.New()
    deferredEvents := list.New()

    // test events
    eventA := new(Event);
    eventA.desc = "test";
    eventA.val = 5;
    events.PushBack(eventA);

    // network elements
    consumer := consumer_Create("consumer1");
    msg := Interest_CreateSimple("lci:/foo/bar");
    consumer.SendInterest(msg);

    nodes := list.New()
    nodes.PushBack(consumer);

    for i := 1; i <= simulationTime; i++ {
        // fmt.Printf("Time = %d...\n", i);

        for e := events.Front(); e != nil; e = e.Next() {
            events.Remove(e);
            event := e.Value.(*Event);

            fmt.Printf("Time %d Event = %d\n", i, event.val);

            if event.val > 0 {
                eventB := new(Event)
                eventB.desc = event.desc
                eventB.val = event.val - 1
                deferredEvents.PushBack(eventB);
            }
        }

        // move each node along
        for e := nodes.Front(); e != nil; e = e.Next() {
            nodeValue := e.Value;

            // type assertions...
            // TODO: handle this cascade type check better
            node, ok := nodeValue.(*Consumer);
            if !ok {
                node, ok := nodeValue.(*Producer);
                if !ok {
                    node, ok := nodeValue.(*Router);
                    if !ok {
                        fmt.Println("ERROR!");
                    } else {
                        node.Tick(i);
                    }
                } else {
                    node.Tick(i);
                }
            } else {
                node.Tick(i);
            }
        }

        // ping pong event queues
        tmp := events;
        events = deferredEvents;
        deferredEvents = tmp;
    }
}

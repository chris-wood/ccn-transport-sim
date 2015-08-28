package main

import "fmt"
import "os"
import "strconv"
import "container/list"
// import "encoding/json"

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

func (i Interest) GetName() (string) {
    return i.Name;
}

func (i Interest) GetPayload() ([]uint8) {
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

func (d Data) GetName() (string) {
    return d.Name;
}

func (d Data) GetPayload() ([]uint8) {
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

type StagedMessage interface {
    GetMessage() Message;
    Ticks() int;
}

type TransmittingMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
    Target int `json:"target"`
}

func (sm TransmittingMessage) Ticks() (int) {
    return sm.TicksLeft;
}

func (sm TransmittingMessage) GetMessage() (Message) {
    return sm.Msg;
}

func (sm TransmittingMessage) GetTarget() (int) {
    return sm.Target;
}

type QueuedMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
    TargetFace int `json:"targetface"`
}

func (qm QueuedMessage) Ticks() (int) {
    return qm.TicksLeft;
}

func (qm QueuedMessage) GetMessage() (Message) {
    return qm.Msg;
}

func (qm QueuedMessage) GetTarget() (int) {
    return qm.TargetFace;
}

type fibentry struct {
    Prefix string `json:"prefix"`
    Interfaces []int `json:"interface"`
}

type fibtable struct {
    Table map[string]fibentry `json:"table"`
}

type pitentry struct {
    Name string `json:"name"`
    Records []Interest `json:"records"`
}

type pittable struct {
    Table map[string]pitentry `json:"table"`
}

type cache struct {
    Cache map[string]Data `json:"cache"`
}

type link struct {
    Stage chan StagedMessage `json:"stage"`
    Capacity int `json:"capacity"`
    Pfail float32 `json:"pfail"`
    Rate float32 `json:"rate"`
}

func (l link) txTime(size int) (int) {
    pktTime := int(float32(size) / l.Rate);
    return pktTime;
}

type LinkFullError struct {
    desc string
}

func (lfe LinkFullError) Error() (string) {
    return lfe.desc;
}

func (l link) PushBack(msg StagedMessage) (error) {
    if (len(l.Stage) < l.Capacity) {
        l.Stage <- msg;
        return nil;
    } else {
        return LinkFullError{"Link full"};
    }
}

type queue struct {
    Fifo chan Message `json:"fifo"`
    Capacity int `json:"size"`
}

type QueueFullError struct {
    desc string
}

func (qfe QueueFullError) Error() (string) {
    return qfe.desc;
}

func (q queue) PushBack(msg Message) (error) {
    if (len(q.Fifo) < q.Capacity) {
        q.Fifo <- msg;
        return nil;
    } else {
        return QueueFullError{"Queue full"};
    }
}

type Forwarder struct {
    Identity string

    Faces []int
    FaceQueues map[int]queue
    FaceLinks map[int]link
    ProcessingPackets chan QueuedMessage

    Fib *fibtable
    Cache *cache
    Pit *pittable
}

type Consumer struct {
    *Forwarder
}

type Runnable interface {
    Tick(int)
}

func (c Consumer) Tick(time int) {

    fmt.Printf("Tick = %d\n", time);

    // Handle links
    for i := 0; i < len(c.Faces); i++ {

        faceId := c.Faces[i];
        link := c.FaceLinks[faceId];

        for j := 0; j < len(link.Stage); j++ {
            msg := <- link.Stage;
            if (msg.GetMessage() != nil) {
                if (msg.Ticks() > 0) {
                    link.Stage <- TransmittingMessage{msg.GetMessage(), msg.Ticks() - 1, 0};
                } else {
                    fmt.Println("OK")
                    // TODO: drop it in the recipient queue here...
                }
            }
        }
    }

    // Handle message processing (one per tick)
    if len(c.ProcessingPackets) > 0 {
        msg := <- c.ProcessingPackets;
        if (msg.Ticks() > 0) {
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTarget()};
            c.ProcessingPackets <- newMsg;
        } else {
            // ticksLeft := link.txTime(len(msg.GetPayload()));
            fmt.Println("okay mang");
            ticksLeft := 2;
            link := c.FaceLinks[msg.GetTarget()];
            stagedMsg := TransmittingMessage{msg.GetMessage(), ticksLeft, 0};
            err := link.PushBack(stagedMsg);
            if err != nil {
                fmt.Println("wtf!");
            }
        }
    }

    // Handle queue movement
    for i := 0; i < len(c.Faces); i++ { // length == 1
        faceId := c.Faces[i];
        queue := c.FaceQueues[faceId];

        for j := 0; j < len(queue.Fifo); j++ {
            msg := <- queue.Fifo;
            if (msg == nil) {
                break;
            }

            // Throw into the processing packet channel for processing delay
            processingTime := 2;
            queuedMessage := QueuedMessage{msg, processingTime, faceId};
            c.ProcessingPackets <- queuedMessage
        }
    }
}

func (c Consumer) SendInterest(msg Interest) {
    defaultFace := c.Faces[0];
    queue := c.FaceQueues[defaultFace];
    err := queue.PushBack(msg);
    if (err != nil) {
        fmt.Println("WTF!");
    }
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
    faceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    fifo := queue{make(chan Message, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    faceMap[defaultFace] = fifo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace}, faceMap, faceLinkMap, processingPackets, &fibtable{}, &cache{}, &pittable{}};
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

    nodes := make([]Runnable, 1);
    nodes[0] = consumer;

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
        for e := 0; e < len(nodes); e++ {
            node := nodes[e];
            node.Tick(i);
        }

        // ping pong event queues
        tmp := events;
        events = deferredEvents;
        deferredEvents = tmp;
    }
}

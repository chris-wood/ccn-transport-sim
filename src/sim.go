package main

import "fmt"
import "os"
import "strconv"
import "container/list"
// import "encoding/json"

type Message interface {
    GetName() string
    GetPayload() []uint8
    ProcessAtRouter(router Router)
    ProcessAtConsumer(consumer Consumer)
    ProcessAtProducer(producer Producer)
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

func (i Interest) ProcessAtRouter(router Router) {
    fmt.Println("router processing interest");
}

func (i Interest) ProcessAtConsumer(consumer Consumer) {
    fmt.Println("consumer processing interest");
}

func (i Interest) ProcessAtProducer(producer Producer) {
    fmt.Println("producer processing interest");
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

func (d Data) ProcessAtRouter(router Router) {
    fmt.Println("router processing data");
}

func (d Data) ProcessAtConsumer(consumer Consumer) {
    fmt.Println("consumer processing data");
}

func (d Data) ProcessAtProducer(producer Producer) {
    fmt.Println("producer processing data");
}

func Data_CreateSimple(name string) (Data) {
    Data := Data{Name: name, Payload: []uint8{0x00}};
    return Data;
}

type Manifest struct {
    Name string `json:"name"`
    Contents []Data `json:"contents"`
}

func (m Manifest) ProcessAtRouter(router Router) {
    fmt.Println("router processing manifest");
}

func (m Manifest) ProcessAtConsumer(consumer Consumer) {
    fmt.Println("consumer processing manifest");
}

func (m Manifest) ProcessAtProducer(producer Producer) {
    fmt.Println("producer processing manifest");
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
    // TargetName string `json:"targetName"`
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
    // TargetName string `json:"targetName"`
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
        fmt.Println("pushing message");
        q.Fifo <- msg;
        return nil;
    } else {
        fmt.Println("error...")
        return QueueFullError{"Queue full"};
    }
}

type Forwarder struct {
    Identity string

    Faces []int
    OutputFaceQueues map[int]queue
    InputFaceQueues map[int]queue
    FaceLinks map[int]link
    FaceLinkQueues map[int]queue
    ProcessingPackets chan QueuedMessage

    Fib *fibtable
    Cache *cache
    Pit *pittable
}

type Consumer struct {
    Fwd *Forwarder
}

type Runnable interface {
    Tick(int)
}

func (f *Forwarder) Tick(time int, upward chan Message) {
    // Handle link delays initiated from this node
    for i := 0; i < len(f.Faces); i++ {

        faceId := f.Faces[i];
        link := f.FaceLinks[faceId];

        for j := 0; j < len(link.Stage); j++ {
            msg := <- link.Stage;
            if (msg.GetMessage() != nil) {
                if (msg.Ticks() > 0) {
                    link.Stage <- TransmittingMessage{msg.GetMessage(), msg.Ticks() - 1, 0};
                } else {
                    fmt.Println("OK")
                    queue := f.FaceLinkQueues[faceId];
                    fmt.Println(queue);
                    queue.PushBack(msg.GetMessage());
                }
            }
        }
    }

    // Handle message processing (one per tick)
    if len(f.ProcessingPackets) > 0 {
        msg := <- f.ProcessingPackets;
        if (msg.Ticks() > 0) {
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTarget()};
            f.ProcessingPackets <- newMsg;
        } else {
            // ticksLeft := link.txTime(len(msg.GetPayload()));
            ticksLeft := 2;
            link := f.FaceLinks[msg.GetTarget()];
            stagedMsg := TransmittingMessage{msg.GetMessage(), ticksLeft, 0};
            err := link.PushBack(stagedMsg);
            if err != nil {
                fmt.Println("wtf!");
            }
        }
    }

    // Handle input queue movement
    for i := 0; i < len(f.Faces); i++ {
        faceId := f.Faces[i];
        queue := f.InputFaceQueues[faceId];

        for j := 0; j < len(queue.Fifo); j++ {
            msg := <- queue.Fifo;
            if (msg == nil) {
                break;
            }

            go func() {
                upward <- msg;
            }();
        }
    }

    // Handle output queue movement
    for i := 0; i < len(f.Faces); i++ { // length == 1
        faceId := f.Faces[i];
        queue := f.OutputFaceQueues[faceId];

        for j := 0; j < len(queue.Fifo); j++ {
            msg := <- queue.Fifo;
            if (msg == nil) {
                break;
            }

            // Throw into the processing packet channel for processing delay
            processingTime := 2;
            queuedMessage := QueuedMessage{msg, processingTime, faceId};
            f.ProcessingPackets <- queuedMessage
        }
    }
}

func (c Consumer) Tick(time int) {
    channel := make(chan Message);
    c.Fwd.Tick(time, channel);
    if len(channel) > 0 {
        for msg := range(channel) {
            msg.ProcessAtConsumer(c);
        }
    }
}

func (c Consumer) SendInterest(msg Interest) {
    defaultFace := c.Fwd.Faces[0];
    queue := c.Fwd.OutputFaceQueues[defaultFace];
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
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    ofifo := queue{make(chan Message, 100), 10};
    ififo := queue{make(chan Message, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, processingPackets, &fibtable{}, &cache{}, &pittable{}};
    consumer := &Consumer{fwd};
    return consumer;
}

type Producer struct {
    Fwd *Forwarder
}

func (p Producer) Tick(time int) {
    channel := make(chan Message);
    p.Fwd.Tick(time, channel);

    if len(channel) > 0 {
        for msg := range(channel) {
            msg.ProcessAtProducer(p)
        }
    }
}

// TODO: how to make queue creation more flexible?

func producer_Create(id string) (*Producer) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    ififo := queue{make(chan Message, 100), 10};
    ofifo := queue{make(chan Message, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, processingPackets, &fibtable{}, &cache{}, &pittable{}};
    producer := &Producer{fwd};
    return producer;
}

type Router struct {
    Fwd *Forwarder
}

func (r Router) Tick(time int) {
    channel := make(chan Message);
    r.Fwd.Tick(time, channel);

    if len(channel) > 0 {
        for msg := range channel {
            msg.ProcessAtRouter(r);
        }
    }
}

func router_Create(id string) (*Router) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    ofifo := queue{make(chan Message, 100), 10};
    ififo := queue{make(chan Message, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, processingPackets, &fibtable{}, &cache{}, &pittable{}};
    router := &Router{fwd};
    return router;
}

type Event struct {
    desc string
    val int
}

func connect(fwd1 *Forwarder, face1 int, fwd2 *Forwarder, face2 int) {
    // face1 --> link1 --> face2 (input queue)
    fwd1.OutputFaceQueues[face1] = fwd1.OutputFaceQueues[1]
    if _, ok := fwd2.InputFaceQueues[face2]; !ok {
        // fmt.Println("adding channels");
        fwd2.InputFaceQueues[face2] = queue{make(chan Message, 100), 10};
    }
    fwd1.FaceLinkQueues[face1] = fwd2.InputFaceQueues[face2];
    // fmt.Println(fwd1.FaceLinkQueues[face1]);

    // face2 --> link2 --> face1 (input queue)
    fwd2.OutputFaceQueues[face2] = fwd2.OutputFaceQueues[1]
    if _, ok := fwd1.InputFaceQueues[face1]; !ok {
        fwd1.InputFaceQueues[face1] = queue{make(chan Message, 100), 10};
    }
    fwd2.FaceLinkQueues[face2] = fwd1.InputFaceQueues[face1];
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

    producer := producer_Create("producer1");
    router := router_Create("router1");

    nodes := make([]Runnable, 3);
    nodes[0] = consumer;
    nodes[1] = router;
    nodes[2] = producer;

    // Make some connections
    connect(consumer.Fwd, 1, router.Fwd, 1);
    connect(router.Fwd, 1, producer.Fwd, 1);

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

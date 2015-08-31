package main

import "fmt"
import "os"
import "strconv"
import "container/list"
import "strings"
// import "encoding/json"

type Message interface {
    GetName() string
    GetPayload() []uint8
    ProcessAtRouter(router Router, face int)
    ProcessAtConsumer(consumer Consumer, face int)
    ProcessAtProducer(producer Producer, face int)
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

func (i Interest) ProcessAtRouter(router Router, arrivalFace int) {
    fmt.Println("router processing interest");

    // TODO: PIT lookup and insertion (for backwards traversal)
    name := i.GetName();
    if router.Fwd.Pit.IsPending(name) {
        router.Fwd.Pit.AddEntry(i);
    } else {
        // Insert into the PIT
        router.Fwd.Pit.AddEntry(i);

        // Forward along
        faces, err := router.Fwd.Fib.GetInterfacesForPrefix(i.GetName());
        if err == nil {
            targetFace := faces[0];
            router.SendInterest(i, arrivalFace, targetFace); // strategy: first record in the longest FIB entry
        } else {
            fmt.Println(err.Error());
            // NOT IN FIB, drop.
        }
    }
}

func (i Interest) ProcessAtConsumer(consumer Consumer, arrivalFace int) {
    fmt.Println("consumer processing interest");
}

func (i Interest) ProcessAtProducer(producer Producer, arrivalFace int) {
    fmt.Println("producer processing interest");
    fmt.Println(arrivalFace);
    data := Data_CreateSimple(i.GetName(), i.GetPayload());

    producer.SendData(data, arrivalFace);
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

func (d Data) ProcessAtRouter(router Router, face int) {
    fmt.Println("router processing data");
    name := d.GetName();
    if router.Fwd.Pit.IsPending(name) {
        entries := router.Fwd.Pit.GetEntries(name);
        router.Fwd.Pit.RemoveEntry(name);

        // forward to all entries...
        for entry := range(entries) {
            fmt.Println(entry);
        }
    }
}

func (d Data) ProcessAtConsumer(consumer Consumer, face int) {
    fmt.Println("consumer processing data");
}

func (d Data) ProcessAtProducer(producer Producer, face int) {
    fmt.Println("producer processing data");
}

func Data_CreateSimple(name string, payload []uint8) (Data) {
    Data := Data{Name: name, Payload: payload};
    return Data;
}

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
    // TODO: make this a real Manifest...
    return Manifest{};
}

type StagedMessage interface {
    GetMessage() Message;
    Ticks() int;
    GetArrivalFace() int;
}

type TransmittingMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
    Target int `json:"target"`
    // TargetName string `json:"targetName"`
}

func (tm TransmittingMessage) GetArrivalFace() (int) {
    return tm.Target;
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
    ArrivalFace int `json:"arrivalface"`
    // TargetName string `json:"targetName"`
}

func (qm QueuedMessage) GetArrivalFace() (int) {
    return qm.ArrivalFace;
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

func (f fibtable) AddPrefix(prefix string, face int) {
    if val, ok := f.Table[prefix]; ok {
        // TODO: face might already be in the interface list
        entry := fibentry{prefix, append(val.Interfaces, face)};
        f.Table[prefix] = entry;
    } else {
        entry := fibentry{prefix, []int{face}};
        f.Table[prefix] = entry;
    }
}

type NotInFibError struct {
    desc string
}

func (nif NotInFibError) Error() (string) {
    return nif.desc;
}

func (f fibtable) GetInterfacesForPrefix(prefix string) ([]int, error){
    var faces []int;

    components := strings.Split(prefix, "/");
    foundEntry := false;

    // Check via LPM
    for i := 0; i < len(components); i++ {
        fullPrefix := ""
        for j := 0; j < i; j++ {
            fullPrefix = fullPrefix + components[j];
            fullPrefix = fullPrefix + "/";
        }
        fullPrefix = fullPrefix + components[i];

        if val, ok := f.Table[fullPrefix]; ok {
            faces = val.Interfaces;
            foundEntry = true;
        }
    }

    if !foundEntry {
        return faces, NotInFibError{fmt.Sprintf("%s not in FIB", prefix)};
    } else {
        return faces, nil;
    }
}

type pitentry struct {
    Name string `json:"name"`
    Records []Interest `json:"records"`
}

type pittable struct {
    Table map[string]pitentry `json:"table"`
}

func (p pittable) RemoveEntry(name string) {
    // _, ok := p.Table[name];
    // assert(ok); // hmm...
    delete(p.Table, name);
}

func (p pittable) GetEntries(name string) ([]Interest) {
    val, _ := p.Table[name];
    // assert(ok); // hmm...
    return val.Records;
}

func (p pittable) AddEntry(msg Interest) {
    name := msg.GetName();
    if val, ok := p.Table[name]; ok {
        entry := pitentry{msg.GetName(), append(val.Records, msg)};
        p.Table[name] = entry;
    } else {
        entry := pitentry{msg.GetName(), []Interest{msg}};
        p.Table[name] = entry;
    }
}

func (p pittable) IsPending(name string) (bool) {
    _, ok := p.Table[name];
    return ok;
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
    Fifo chan StagedMessage `json:"fifo"`
    Capacity int `json:"size"`
}

type QueueFullError struct {
    desc string
}

func (qfe QueueFullError) Error() (string) {
    return qfe.desc;
}

func (q queue) PushBackQueuedMessage(msg StagedMessage) (error) {
    if (len(q.Fifo) < q.Capacity) {
        q.Fifo <- msg;
        return nil;
    } else {
        fmt.Println("error...")
        return QueueFullError{"Queue full"};
    }
}


func (q queue) PushBack(msg Message) (error) {
    if (len(q.Fifo) < q.Capacity) {
        stagedMessage := QueuedMessage{msg, 0, 0, 0};
        q.PushBackQueuedMessage(stagedMessage);
        // q.Fifo <- msg;
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
    FaceToFace map[int]int
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

func (f *Forwarder) Tick(time int, upward chan StagedMessage, doneChannel chan int) {
    // Handle link delays initiated from this node
    for i := 0; i < len(f.Faces); i++ {

        faceId := f.Faces[i];
        link := f.FaceLinks[faceId];

        for j := 0; j < len(link.Stage); j++ {
            msg := <- link.Stage;
            if (msg.GetMessage() != nil) {
                if (msg.Ticks() > 0) {
                    link.Stage <- TransmittingMessage{msg.GetMessage(), msg.Ticks() - 1, f.FaceToFace[faceId]};
                } else {
                    fmt.Println("OK -- putting in recipient queue");
                    queue := f.FaceLinkQueues[faceId];
                    queue.PushBackQueuedMessage(msg);
                }
            }
        }
    }

    // Handle message processing (one per tick)
    if len(f.ProcessingPackets) > 0 {
        msg := <- f.ProcessingPackets;
        if (msg.Ticks() > 0) {
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTarget(), msg.GetArrivalFace()};
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
    // sent := false
    for i := 0; i < len(f.Faces); i++ {
        faceId := f.Faces[i];
        queue := f.InputFaceQueues[faceId];

        for j := 0; j < len(queue.Fifo); j++ {
            msg := <- queue.Fifo;
            if (msg == nil) {
                break;
            }
            upward <- msg;
            // sent = true;
        }
    }
    upward <- nil; // signal completion

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
            queuedMessage := QueuedMessage{msg.GetMessage(), processingTime, faceId, msg.GetArrivalFace()};
            f.ProcessingPackets <- queuedMessage
        }
    }

    // signal completion.
    doneChannel <- 1;
}

func (c Consumer) Tick(time int) {
    channel := make(chan StagedMessage);
    doneChannel := make(chan int);
    doneUpwardsProcessing := make(chan int);

    go c.Fwd.Tick(time, channel, doneChannel);

    go func() {
        for {
            msg := <-channel;
            if msg == nil {
                break;
            }
            fmt.Println("processing at consumer...")
            realMsg := msg.GetMessage();
            realMsg.ProcessAtConsumer(c, msg.GetArrivalFace());
        };
        doneUpwardsProcessing <- 1;
    }();

    // block until completion
    <-doneChannel;
    <-doneUpwardsProcessing;
}

func (c Consumer) SendInterest(msg Interest) {

    defaultFace := c.Fwd.Faces[0];
    queue := c.Fwd.OutputFaceQueues[defaultFace];

    // stagedMsg := QueuedMessage{msg, 0, 0, -1};
    err := queue.PushBack(msg);
    // err := queue.PushBack(stagedMsg);
    if (err != nil) {
        fmt.Println("WTF!");
    }
}

func consumer_Create(id string) (*Consumer) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ofifo := queue{make(chan StagedMessage, 100), 10};
    ififo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &fibtable{Table: make(map[string]fibentry)}, &cache{}, &pittable{}};
    consumer := &Consumer{fwd};
    return consumer;
}

type Producer struct {
    Fwd *Forwarder
}

func (p Producer) Tick(time int) {
    channel := make(chan StagedMessage);
    doneChannel := make(chan int);
    doneUpwardsProcessing := make(chan int);

    go p.Fwd.Tick(time, channel, doneChannel);
    go func() {
        for {
            msg := <-channel;
            if msg == nil {
                break;
            }
            networkMessage := msg.GetMessage();
            networkMessage.ProcessAtProducer(p, msg.GetArrivalFace());
        };
        doneUpwardsProcessing <- 1;
    }();

    <-doneChannel; // wait until the forwarder is done working
    <-doneUpwardsProcessing; // wait until we're done processing the upper-layer messages
}

func (p Producer) SendData(msg Data, targetFace int) {
    queue := p.Fwd.OutputFaceQueues[targetFace];
    err := queue.PushBack(msg);
    if (err != nil) {
        fmt.Println(err.Error());
    }
}

// TODO: how to make queue creation more flexible?

func producer_Create(id string) (*Producer) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ififo := queue{make(chan StagedMessage, 100), 10};
    ofifo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &fibtable{Table: make(map[string]fibentry)}, &cache{}, &pittable{}};
    producer := &Producer{fwd};
    return producer;
}

type Router struct {
    Fwd *Forwarder
}

func (r Router) Tick(time int) {
    channel := make(chan StagedMessage);
    doneChannel := make(chan int);
    doneUpwardsProcessing := make(chan int);

    go r.Fwd.Tick(time, channel, doneChannel);
    go func() {
        for {
            msg := <-channel;
            if msg == nil {
                break;
            }
            fmt.Println("processing...")
            // msg.ProcessAtRouter(r);
            realMsg := msg.GetMessage();
            realMsg.ProcessAtRouter(r, msg.GetArrivalFace());
        };
        doneUpwardsProcessing <- 1;
    }();

    <-doneChannel;
    <-doneUpwardsProcessing;
}

func (r Router) SendInterest(msg Interest, arrivalFace int, targetFace int) {
    fmt.Println("Router sending interest");
    queue := r.Fwd.OutputFaceQueues[targetFace];
    err := queue.PushBackQueuedMessage(QueuedMessage{msg, 0, arrivalFace, targetFace});
    if (err != nil) {
        fmt.Println("WTF!");
    }
}

func router_Create(id string) (*Router) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ofifo := queue{make(chan StagedMessage, 100), 10};
    ififo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &fibtable{Table: make(map[string]fibentry)}, &cache{}, &pittable{}};
    router := &Router{fwd};
    return router;
}

type Event struct {
    desc string
    val int
}

func connect(fwd1 *Forwarder, face1 int, fwd2 *Forwarder, face2 int, prefix string) {
    // face1 --> link1 --> face2 (input queue)
    fwd1.OutputFaceQueues[face1] = fwd1.OutputFaceQueues[1]
    if _, ok := fwd2.InputFaceQueues[face2]; !ok {
        fwd2.InputFaceQueues[face2] = queue{make(chan StagedMessage, 100), 10};
    }
    fwd1.FaceLinkQueues[face1] = fwd2.InputFaceQueues[face2];

    // face2 --> link2 --> face1 (input queue)
    fwd2.OutputFaceQueues[face2] = fwd2.OutputFaceQueues[1]
    if _, ok := fwd1.InputFaceQueues[face1]; !ok {
        fwd1.InputFaceQueues[face1] = queue{make(chan StagedMessage, 100), 10};
    }
    fwd2.FaceLinkQueues[face2] = fwd1.InputFaceQueues[face1];

    // face connection
    fwd1.FaceToFace[face1] = face2;
    fwd2.FaceToFace[face2] = face1;

    // prefix configuration
    fwd1.Fib.AddPrefix(prefix, face1);
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
    msg := Interest_CreateSimple("/foo/bar");
    consumer.SendInterest(msg);

    producer := producer_Create("producer1");
    router := router_Create("router1");

    nodes := make([]Runnable, 3);
    nodes[0] = consumer;
    nodes[1] = router;
    nodes[2] = producer;

    // Make some connections
    connect(consumer.Fwd, 1, router.Fwd, 1, "/foo");
    connect(router.Fwd, 1, producer.Fwd, 1, "/foo");

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

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
    // fmt.Println("router processing interest");

    // TODO: PIT lookup and insertion (for backwards traversal)
    name := i.GetName();
    if router.Fwd.Pit.IsPending(name) {
        router.Fwd.Pit.AddEntry(i, arrivalFace);
    } else {
        // Insert into the PIT
        router.Fwd.Pit.AddEntry(i, arrivalFace);

        // Forward along
        faces, err := router.Fwd.Fib.GetInterfacesForPrefix(i.GetName());
        if err == nil {
            targetFace := faces[0];
            // fmt.Printf("forwarding from %d to %d\n", arrivalFace, targetFace);
            router.SendInterest(i, arrivalFace, targetFace); // strategy: first record in the longest FIB entry
        } else {
            fmt.Println(err.Error());
            // NOT IN FIB, drop.
        }
    }
}

func (i Interest) ProcessAtConsumer(consumer Consumer, arrivalFace int) {
    // fmt.Println("consumer processing interest");
}

func (i Interest) ProcessAtProducer(producer Producer, arrivalFace int) {
    // fmt.Printf("producer processing interest from face %d\n", arrivalFace);
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

func (d Data) ProcessAtRouter(router Router, arrivalFace int) {
    // fmt.Println("router processing data");
    name := d.GetName();
    if router.Fwd.Pit.IsPending(name) {
        entries := router.Fwd.Pit.GetEntries(name);
        router.Fwd.Pit.RemoveEntry(name);

        // forward to all entries...
        for i := 0; i < len(entries); i++ {
            entry := entries[i];
            targetFace := entry.arrivalFace;
            router.SendData(d, arrivalFace, targetFace); // strategy: first record in the longest FIB entry
        }
    } else {
        fmt.Println("not pending!!");
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
    return Manifest{};
}

type StagedMessage interface {
    GetMessage() Message;
    Ticks() int;
    GetArrivalFace() int;
    GetTargetFace() int;
}

type QueuedMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
    TargetFace int `json:"targetface"`
    ArrivalFace int `json:"arrivalface"`
}

func (qm QueuedMessage) GetArrivalFace() (int) {
    return qm.ArrivalFace;
}

func (qm QueuedMessage) GetTargetFace() (int) {
    return qm.TargetFace;
}

func (qm QueuedMessage) Ticks() (int) {
    return qm.TicksLeft;
}

func (qm QueuedMessage) GetMessage() (Message) {
    return qm.Msg;
}

type FibTableEntry struct {
    Prefix string `json:"prefix"`
    Interfaces []int `json:"interface"`
}

type FibTable struct {
    Table map[string]FibTableEntry `json:"table"`
}

func (f FibTable) AddPrefix(prefix string, face int) {
    if val, ok := f.Table[prefix]; ok {
        // TODO: face might already be in the interface list
        entry := FibTableEntry{prefix, append(val.Interfaces, face)};
        f.Table[prefix] = entry;
    } else {
        entry := FibTableEntry{prefix, []int{face}};
        f.Table[prefix] = entry;
    }
}

type NotInFibError struct {
    desc string
}

func (nif NotInFibError) Error() (string) {
    return nif.desc;
}

func (f FibTable) GetInterfacesForPrefix(prefix string) ([]int, error){
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

type PitEntryRecord struct {
    arrivalFace int
    msg Interest
}

type PitEntry struct {
    Name string `json:"name"`
    Records []PitEntryRecord `json:"records"`
}

type PitTable struct {
    Table map[string]PitEntry `json:"table"`
}

func (p PitTable) RemoveEntry(name string) {
    // _, ok := p.Table[name];
    // assert(ok); // hmm...
    delete(p.Table, name);
}

func (p PitTable) GetEntries(name string) ([]PitEntryRecord) {
    val, _ := p.Table[name];
    // assert(ok); // hmm...
    return val.Records;
}

// TODO: left off here -- fixing the PitEntry contents
func (p PitTable) AddEntry(msg Interest, arrivalFace int) {
    name := msg.GetName();
    if val, ok := p.Table[name]; ok {
        record := PitEntryRecord{arrivalFace, msg};
        entry := PitEntry{msg.GetName(), append(val.Records, record)};
        p.Table[name] = entry;
    } else {
        records := make([]PitEntryRecord, 1);
        records[0] = PitEntryRecord{arrivalFace, msg};
        entry := PitEntry{msg.GetName(), records};
        p.Table[name] = entry;
    }
}

func (p PitTable) IsPending(name string) (bool) {
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
        return QueueFullError{"Queue full"};
    }
}


func (q queue) PushBack(msg Message, arrivalFace int, targetFace int) (error) {
    if (len(q.Fifo) < q.Capacity) {
        stagedMessage := QueuedMessage{msg, 0, targetFace, arrivalFace};
        q.PushBackQueuedMessage(stagedMessage);
        return nil;
    } else {
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

    Fib *FibTable
    Cache *cache
    Pit *PitTable
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
                    link.Stage <- QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTargetFace(), msg.GetArrivalFace()};
                } else {
                    queue := f.FaceLinkQueues[msg.GetArrivalFace()];
                    queue.PushBackQueuedMessage(msg);
                }
            }
        }
    }

    // Handle message processing (one per tick)
    if len(f.ProcessingPackets) > 0 {
        msg := <- f.ProcessingPackets;
        if (msg.Ticks() > 0) {
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTargetFace(), msg.GetArrivalFace()};
            f.ProcessingPackets <- newMsg;
        } else {
            // ticksLeft := link.txTime(len(msg.GetPayload()));
            ticksLeft := 2;
            link := f.FaceLinks[msg.GetTargetFace()];
            stagedMsg := QueuedMessage{msg.GetMessage(), ticksLeft, f.FaceToFace[msg.GetTargetFace()], msg.GetTargetFace()};
            err := link.PushBack(stagedMsg);
            if err != nil {
                fmt.Println(err.Error());
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
            upward <- msg;
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
            queuedMessage := QueuedMessage{msg.GetMessage(), processingTime, msg.GetTargetFace(), msg.GetArrivalFace()};
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
            realMsg := msg.GetMessage();
            realMsg.ProcessAtConsumer(c, msg.GetArrivalFace()); // the arrival face is the arrival face
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
    arrivalFace := defaultFace;
    targetFace := defaultFace;
    err := queue.PushBack(msg, arrivalFace, targetFace);
    if (err != nil) {
        fmt.Println(err.Error());
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

    fwd := &Forwarder{id, []int{defaultFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &FibTable{Table: make(map[string]FibTableEntry)}, &cache{},
        &PitTable{Table: make(map[string]PitEntry)}};
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
            networkMessage.ProcessAtProducer(p, msg.GetTargetFace()); // the arrival face is the same as the target face
        };
        doneUpwardsProcessing <- 1;
    }();

    <-doneChannel; // wait until the forwarder is done working
    <-doneUpwardsProcessing; // wait until we're done processing the upper-layer messages
}

func (p Producer) SendData(msg Data, targetFace int) {
    queue := p.Fwd.OutputFaceQueues[targetFace];
    err := queue.PushBack(msg, 1, targetFace);
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

    fwd := &Forwarder{id, []int{defaultFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &FibTable{Table: make(map[string]FibTableEntry)}, &cache{},
        &PitTable{Table: make(map[string]PitEntry)}};
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
            realMsg := msg.GetMessage();
            realMsg.ProcessAtRouter(r, msg.GetArrivalFace()); // the arrival face is... the arrival face
        };
        doneUpwardsProcessing <- 1;
    }();

    <-doneChannel;
    <-doneUpwardsProcessing;
}

func (r Router) SendInterest(msg Interest, arrivalFace int, targetFace int) {
    queue := r.Fwd.OutputFaceQueues[targetFace];
    err := queue.PushBackQueuedMessage(QueuedMessage{msg, 0, targetFace, arrivalFace});
    if (err != nil) {
        fmt.Println(err.Error());
    }
}

func (r Router) SendData(msg Data, arrivalFace int, targetFace int) {
    queue := r.Fwd.OutputFaceQueues[targetFace];
    err := queue.PushBackQueuedMessage(QueuedMessage{msg, 0, targetFace, arrivalFace});
    if (err != nil) {
        fmt.Println(err.Error());
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
        &FibTable{Table: make(map[string]FibTableEntry)}, &cache{},
        &PitTable{make(map[string]PitEntry)}};
    router := &Router{fwd};
    return router;
}

type Event struct {
    desc string
    val int
}

func connect(fwd1 *Forwarder, face1 int, fwd2 *Forwarder, face2 int, prefix string) {
    // face1 --> link1 --> face2 (input queue)
    fwd1.OutputFaceQueues[face1] = fwd1.OutputFaceQueues[1] // all queues dump to the default output queue
    if _, ok := fwd2.InputFaceQueues[face2]; !ok {
        fwd2.InputFaceQueues[face2] = queue{make(chan StagedMessage, 100), 10};
    }
    if _, ok := fwd1.FaceLinks[face1]; !ok {
        link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
        fwd1.FaceLinks[face1] = link;
    }
    fwd1.FaceLinkQueues[face1] = fwd2.InputFaceQueues[face2];

    // face2 --> link2 --> face1 (input queue)
    fwd2.OutputFaceQueues[face2] = fwd2.OutputFaceQueues[1] // all queues dump to the default output queue
    if _, ok := fwd1.InputFaceQueues[face1]; !ok {
        fwd1.InputFaceQueues[face1] = queue{make(chan StagedMessage, 100), 10};
    }
    if _, ok := fwd2.FaceLinks[face1]; !ok {
        link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
        fwd2.FaceLinks[face2] = link;
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
    eventA.desc = "Send interest command";
    eventA.val = 5; // every 5 ticks
    events.PushBack(eventA);

    // network elements
    consumer := consumer_Create("consumer1");
    producer := producer_Create("producer1");
    router := router_Create("router1");

    nodes := make([]Runnable, 3);
    nodes[0] = consumer;
    nodes[1] = router;
    nodes[2] = producer;

    // Make some connections
    connect(consumer.Fwd, 1, router.Fwd, 1, "/foo");
    connect(router.Fwd, 2, producer.Fwd, 1, "/foo");

    for i := 1; i <= simulationTime; i++ {

        // event processing pipeline
        for e := events.Front(); e != nil; e = e.Next() {
            events.Remove(e);
            event := e.Value.(*Event);
            if event.val > 0 {
                eventB := new(Event)
                eventB.desc = event.desc
                eventB.val = event.val - 1
                deferredEvents.PushBack(eventB);
            } else {
                msg := Interest_CreateSimple("/foo/bar");
                consumer.SendInterest(msg);
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

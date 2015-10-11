package ccnsim

import "fmt"
import "strings"
import "math/rand"

func makeNonce(length int) (string) {
    runes := []rune("0123456789"); // runes are single-character text encodings by convention
    nonce := make([]rune, length);
    for i := 0; i < length; i++ {
        nonce[i] = runes[rand.Intn(len(runes))];
    }
    return string(nonce);
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

type ContentStore struct {
    Cache map[string]Data `json:"cache"`
}
func (c ContentStore) GetData(name string) (Data) {
    val, _ := c.Cache[name];
    return val;
}

func (c ContentStore) AddData(d Data) {
    if _, ok := c.Cache[d.GetName()]; !ok {
        c.Cache[d.GetName()] = d;
    }
}

func (c ContentStore) HasData(name string) (bool) {
    _, ok := c.Cache[name];
    return ok;
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
        stagedMessage := QueuedMessage{Msg: msg, TicksLeft: 0, TargetFace: targetFace, ArrivalFace: arrivalFace};
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
    Cache *ContentStore
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
                    // arrival: output from sender, so the corresponding queue is the input queue at the receiver
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
            // TODO: need separate queue for ping pong...
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTargetFace(), msg.GetArrivalFace()};
            f.ProcessingPackets <- newMsg;
        } else {
            // ticksLeft := link.txTime(len(msg.GetPayload()));
            ticksLeft := 2;

            // target: input face on receiver
            // arrival: output face on sender
            link := f.FaceLinks[msg.GetArrivalFace()];

            stagedMsg := QueuedMessage{msg.GetMessage(), ticksLeft, msg.GetTargetFace(), msg.GetArrivalFace()};
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

            // NOTE: target face here is the output face for the sender
            //  We need to adjust the face pairs so that the arrival face is the target face,
            //  and the target face is the face
            newTarget := f.FaceToFace[msg.GetTargetFace()];
            newArrival := msg.GetTargetFace();

            queuedMessage := QueuedMessage{Msg: msg.GetMessage(), TicksLeft: processingTime, TargetFace: newTarget, ArrivalFace: newArrival};

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
    appFace := 0;
    queue := c.Fwd.OutputFaceQueues[defaultFace];

    // send interest from LOCAL FACE to DEFAULT FACE
    targetFace := defaultFace;

    err := queue.PushBack(msg, appFace, targetFace);
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
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{},
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
    appFace := 0;
    err := queue.PushBack(msg, appFace, targetFace);
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
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{},
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
            realMsg.ProcessAtRouter(r, msg.GetTargetFace());
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
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{make(map[string]Data)},
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

    // repr, _ := json.Marshal(fwd2.InputFaceQueues);
    // fmt.Println(string(repr));

    // face connection
    fwd1.FaceToFace[face1] = face2;
    fwd2.FaceToFace[face2] = face1;

    // prefix configuration
    fwd1.Fib.AddPrefix(prefix, face1);
}

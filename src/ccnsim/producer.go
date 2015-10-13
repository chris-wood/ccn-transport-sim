package ccnsim

import "fmt"

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

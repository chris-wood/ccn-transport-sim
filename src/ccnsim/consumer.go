package ccnsim

import "fmt"

type Consumer struct {
    Fwd *Forwarder
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

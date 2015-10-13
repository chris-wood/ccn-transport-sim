package ccnsim

import "fmt"

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

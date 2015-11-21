package core

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

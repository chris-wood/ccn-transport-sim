package core

type Producer struct {
    Fwd *Forwarder
    AppFace int
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
            incomingFace := msg.GetDstFace();
            networkMessage.ProcessAtProducer(p, incomingFace);
        };
        doneUpwardsProcessing <- 1;
    }();

    <-doneChannel; // wait until the forwarder is done working
    <-doneUpwardsProcessing; // wait until we're done processing the upper-layer messages
}

func (p Producer) SendData(msg Data, incomingFace int) {
    p.Fwd.AddOutboundMessage(msg, p.AppFace, incomingFace);
}

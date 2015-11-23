package core

type Consumer struct {
    Fwd *Forwarder
    AppFace int
    DefaultFace int
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
            incomingFace := msg.GetDstFace();
            realMsg.ProcessAtConsumer(c, incomingFace);
        };
        doneUpwardsProcessing <- 1;
    }();

    // block until completion
    <-doneChannel;
    <-doneUpwardsProcessing;
}

func (c Consumer) SendInterest(msg Interest) {
//    defaultFace := c.Fwd.Faces[0];
    c.Fwd.AddOutboundMessage(msg, c.AppFace, c.DefaultFace);
}

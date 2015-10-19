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

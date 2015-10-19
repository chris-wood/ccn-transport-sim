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

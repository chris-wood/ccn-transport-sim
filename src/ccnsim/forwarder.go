package ccnsim

import "fmt"

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

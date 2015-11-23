package core

import "fmt"

type Forwarder struct {
    Identity string

    Faces []int // set of all faces
    OutputFaceQueues map[int]Queue // set of outbound queues for faces
    InputFaceQueues map[int]Queue // set of inbound queues for faces
    FaceLinks map[int]Link // map of faces to links
    FaceLinkQueues map[int]Queue // map of faces to recipient queues
    FaceToFace map[int]int // map of fwd1.face to fwd2.face (the actual pair-wise face connection)

    ProcessingPackets chan QueuedMessage

    Fib *FibTable
    Cache *ContentStore
    Pit *PitTable
}

func (f *Forwarder) AddOutboundMessage(msg Message, srcFace int, dstFace int) {
    queue := f.OutputFaceQueues[dstFace];
    err := queue.PushBack(msg, srcFace, dstFace);
    if (err != nil) {
        fmt.Println(err.Error());
    }
}

func (f *Forwarder) processOutboundQueues() {
    // Handle output Queue movement
    for i := 0; i < len(f.Faces); i++ { // length == 1
        faceId := f.Faces[i];
        Queue := f.OutputFaceQueues[faceId];

        for j := 0; j < len(Queue.Fifo); j++ {
            msg := <- Queue.Fifo;
            if (msg == nil) {
                break;
            }

            // Throw into the processing packet channel for processing delay
            processingTime := 2;

            // NOTE: target face here is the output face for the sender
            //  We need to adjust the face pairs so that the arrival face is the target face,
            //  and the target face is the face
            newDst := f.FaceToFace[msg.GetDstFace()];
            newSrc := msg.GetDstFace();

            QueuedMessage := QueuedMessage{Msg: msg.GetMessage(), TicksLeft: processingTime, DstFace: newDst, SrcFace: newSrc};

            f.ProcessingPackets <- QueuedMessage
        }
    }
}

func (f *Forwarder) processInboundQueues(upward chan StagedMessage) {
    for i := 0; i < len(f.Faces); i++ {
        faceId := f.Faces[i];
        Queue := f.InputFaceQueues[faceId];

        for j := 0; j < len(Queue.Fifo); j++ {
            msg := <- Queue.Fifo;
            if (msg == nil) {
                break;
            }
            upward <- msg;
        }
    }
    upward <- nil; // Signal the end of the messages.
}

func (f *Forwarder) Tick(time int, upward chan StagedMessage, doneChannel chan int) {
    // // Handle link delays initiated from this node
    // for i := 0; i < len(f.Faces); i++ {
    //
    //     faceId := f.Faces[i];
    //     link := f.FaceLinks[faceId];
    //
    //     for j := 0; j < len(link.Stage); j++ {
    //         msg := <- link.Stage;
    //         if (msg.GetMessage() != nil) {
    //             if (msg.Ticks() > 0) {
    //                 link.Stage <- QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetTargetFace(), msg.GetArrivalFace()};
    //             } else {
    //                 // arrival: output from sender, so the corresponding Queue is the input Queue at the receiver
    //                 Queue := f.FaceLinkQueues[msg.GetArrivalFace()];
    //                 Queue.PushBackQueuedMessage(msg);
    //             }
    //         }
    //     }
    // }

    // Handle message processing (one per tick)
    if len(f.ProcessingPackets) > 0 {
        msg := <- f.ProcessingPackets;
        if (msg.Ticks() > 0) {
            // TODO: need separate Queue for ping pong
            newMsg := QueuedMessage{msg.GetMessage(), msg.Ticks() - 1, msg.GetDstFace(), msg.GetSrcFace()};
            f.ProcessingPackets <- newMsg;
        } else {
            // ticksLeft := link.txTime(len(msg.GetPayload()));
            ticksLeft := 2;

            // target: input face on receiver
            // arrival: output face on sender
            link := f.FaceLinks[msg.GetSrcFace()];

            stagedMsg := QueuedMessage{msg.GetMessage(), ticksLeft, msg.GetDstFace(), msg.GetSrcFace()};
            err := link.PushBack(stagedMsg);
            if err != nil {
                fmt.Println(err.Error());
            }
        }
    }

    // Handle inbound queue movement
    f.processInboundQueues(upward);

    // Handle outbound queue movement
    f.processOutboundQueues();

    // signal completion.
    doneChannel <- 1;
}

package core

type Queue struct {
    Fifo chan StagedMessage `json:"fifo"`
    Capacity int `json:"size"`
}

type QueueFullError struct {
    desc string
}

func (qfe QueueFullError) Error() (string) {
    return qfe.desc;
}

func (q Queue) PushBackQueuedMessage(msg StagedMessage) (error) {
    if (len(q.Fifo) < q.Capacity) {
        q.Fifo <- msg;
        return nil;
    } else {
        return QueueFullError{"Queue full"};
    }
}


func (q Queue) PushBack(msg Message, srcFace int, dstFace int) (error) {
    if (len(q.Fifo) < q.Capacity) {
        stagedMessage := QueuedMessage{Msg: msg, TicksLeft: 0, SrcFace: srcFace, DstFace: dstFace};
        q.PushBackQueuedMessage(stagedMessage);
        return nil;
    } else {
        return QueueFullError{"Queue full"};
    }
}

package ccnsim

type queue struct {
    Fifo chan StagedMessage `json:"fifo"`
    Capacity int `json:"size"`
}

type QueueFullError struct {
    desc string
}

func (qfe QueueFullError) Error() (string) {
    return qfe.desc;
}

func (q queue) PushBackQueuedMessage(msg StagedMessage) (error) {
    if (len(q.Fifo) < q.Capacity) {
        q.Fifo <- msg;
        return nil;
    } else {
        return QueueFullError{"Queue full"};
    }
}


func (q queue) PushBack(msg Message, arrivalFace int, targetFace int) (error) {
    if (len(q.Fifo) < q.Capacity) {
        stagedMessage := QueuedMessage{Msg: msg, TicksLeft: 0, TargetFace: targetFace, ArrivalFace: arrivalFace};
        q.PushBackQueuedMessage(stagedMessage);
        return nil;
    } else {
        return QueueFullError{"Queue full"};
    }
}
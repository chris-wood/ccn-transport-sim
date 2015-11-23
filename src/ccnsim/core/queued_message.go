package core

type QueuedMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`

    SrcFace int `json:"srcFace"`
    DstFace int `json:"dstFace"`
}

func (qm QueuedMessage) GetSrcFace() (int) {
    return qm.SrcFace;
}

func (qm QueuedMessage) GetDstFace() (int) {
    return qm.DstFace;
}

func (qm QueuedMessage) Ticks() (int) {
    return qm.TicksLeft;
}

func (qm QueuedMessage) GetMessage() (Message) {
    return qm.Msg;
}

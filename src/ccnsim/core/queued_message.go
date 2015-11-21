package core

type QueuedMessage struct {
    Msg Message `json:"msg"`
    TicksLeft int `json:"ticksleft"`
    TargetFace int `json:"targetface"`
    ArrivalFace int `json:"arrivalface"`
}

func (qm QueuedMessage) GetArrivalFace() (int) {
    return qm.ArrivalFace;
}

func (qm QueuedMessage) GetTargetFace() (int) {
    return qm.TargetFace;
}

func (qm QueuedMessage) Ticks() (int) {
    return qm.TicksLeft;
}

func (qm QueuedMessage) GetMessage() (Message) {
    return qm.Msg;
}

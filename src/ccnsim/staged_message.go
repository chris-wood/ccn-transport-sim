package ccnsim

type StagedMessage interface {
    GetMessage() Message;
    Ticks() int;
    GetArrivalFace() int;
    GetTargetFace() int;
}

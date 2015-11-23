package core

type StagedMessage interface {
    GetMessage() Message;
    Ticks() int;
    GetSrcFace() int;
    GetDstFace() int;
}

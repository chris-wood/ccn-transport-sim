package core

type Link struct {
    Stage chan StagedMessage `json:"stage"`
    Capacity int `json:"capacity"`
    Pfail float32 `json:"pfail"`
    Rate float32 `json:"rate"`
}

func (l Link) txTime(size int) (int) {
    pktTime := int(float32(size) / l.Rate);
    return pktTime;
}

type LinkFullError struct {
    desc string
}

func (lfe LinkFullError) Error() (string) {
    return lfe.desc;
}

func (l Link) PushBack(msg StagedMessage) (error) {
    if (len(l.Stage) < l.Capacity) {
        l.Stage <- msg;
        return nil;
    } else {
        return LinkFullError{"Link full"};
    }
}

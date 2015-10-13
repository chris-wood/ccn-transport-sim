package ccnsim

type link struct {
    Stage chan StagedMessage `json:"stage"`
    Capacity int `json:"capacity"`
    Pfail float32 `json:"pfail"`
    Rate float32 `json:"rate"`
}

func (l link) txTime(size int) (int) {
    pktTime := int(float32(size) / l.Rate);
    return pktTime;
}

type LinkFullError struct {
    desc string
}

func (lfe LinkFullError) Error() (string) {
    return lfe.desc;
}

func (l link) PushBack(msg StagedMessage) (error) {
    if (len(l.Stage) < l.Capacity) {
        l.Stage <- msg;
        return nil;
    } else {
        return LinkFullError{"Link full"};
    }
}

package core

type PitEntryRecord struct {
    arrivalFace int
    msg Interest
}

type PitEntry struct {
    Name string `json:"name"`
    Records []PitEntryRecord `json:"records"`
}

type PitTable struct {
    Table map[string]PitEntry `json:"table"`
}

func (p PitTable) RemoveEntry(name string) {
    // _, ok := p.Table[name];
    // assert(ok); // hmm...
    delete(p.Table, name);
}

func (p PitTable) GetEntries(name string) ([]PitEntryRecord) {
    val, _ := p.Table[name];
    // assert(ok); // hmm...
    return val.Records;
}

func (p PitTable) AddEntry(msg Interest, arrivalFace int) {
    name := msg.GetName();
    if val, ok := p.Table[name]; ok {
        record := PitEntryRecord{arrivalFace, msg};
        entry := PitEntry{msg.GetName(), append(val.Records, record)};
        p.Table[name] = entry;
    } else {
        records := make([]PitEntryRecord, 1);
        records[0] = PitEntryRecord{arrivalFace, msg};
        entry := PitEntry{msg.GetName(), records};
        p.Table[name] = entry;
    }
}

func (p PitTable) IsPending(name string) (bool) {
    _, ok := p.Table[name];
    return ok;
}

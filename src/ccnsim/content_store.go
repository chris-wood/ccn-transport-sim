package ccnsim

type ContentStore struct {
    Cache map[string]Data `json:"cache"`
}
func (c ContentStore) GetData(name string) (Data) {
    val, _ := c.Cache[name];
    return val;
}

func (c ContentStore) AddData(d Data) {
    if _, ok := c.Cache[d.GetName()]; !ok {
        c.Cache[d.GetName()] = d;
    }
}

func (c ContentStore) HasData(name string) (bool) {
    _, ok := c.Cache[name];
    return ok;
}

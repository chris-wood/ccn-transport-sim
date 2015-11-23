package core

type Network struct {
    forwarders []Runnable
}

func (n Network) Tick(time int) {
    for _, fwd := range n.forwarders {
        fwd.Tick(time);
    }
}

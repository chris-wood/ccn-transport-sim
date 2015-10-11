package ccnsim

type NetworkMessage struct {
    Nonce string `json:"nonce"`
    HopCount int `json:"hopcount"`
}

func (nm NetworkMessage) GetNonce() (string) {
    return nm.Nonce;
}

func (nm NetworkMessage) GetHopCount() (int) {
    return nm.HopCount;
}

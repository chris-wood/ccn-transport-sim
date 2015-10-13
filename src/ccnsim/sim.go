package ccnsim

import "strings"
import "math/rand"

func makeNonce(length int) (string) {
    runes := []rune("0123456789"); // runes are single-character text encodings by convention
    nonce := make([]rune, length);
    for i := 0; i < length; i++ {
        nonce[i] = runes[rand.Intn(len(runes))];
    }
    return string(nonce);
}

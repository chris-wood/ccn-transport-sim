package ccnsim

import "math/rand"

type Simulator struct {
}

func makeNonce(length int) (string) {
    runes := []rune("0123456789"); // runes are single-character text encodings by convention
    nonce := make([]rune, length);
    for i := 0; i < length; i++ {
        nonce[i] = runes[rand.Intn(len(runes))];
    }
    return string(nonce);
}

// TODO: move these things to a helper class

func (s Simulator) Consumer_Create(id string) (*Consumer) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ofifo := queue{make(chan StagedMessage, 100), 10};
    ififo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{},
        &PitTable{Table: make(map[string]PitEntry)}};
    consumer := &Consumer{fwd};
    return consumer;
}

// TODO: how to make queue creation more flexible?

func (s Simulator) Producer_Create(id string) (*Producer) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ififo := queue{make(chan StagedMessage, 100), 10};
    ofifo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{},
        &PitTable{Table: make(map[string]PitEntry)}};
    producer := &Producer{fwd};
    return producer;
}

func (s Simulator) Router_Create(id string) (*Router) {
    outputFaceMap := make(map[int]queue);
    inputFaceMap := make(map[int]queue);
    faceLinkMap := make(map[int]link);
    faceLinkMapQueues := make(map[int]queue);
    faceToFace := make(map[int]int);
    ofifo := queue{make(chan StagedMessage, 100), 10};
    ififo := queue{make(chan StagedMessage, 100), 10};
    link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &FibTable{Table: make(map[string]FibTableEntry)}, &ContentStore{make(map[string]Data)},
        &PitTable{make(map[string]PitEntry)}};
    router := &Router{fwd};
    return router;
}

func (s Simulator) Connect(fwd1 *Forwarder, face1 int, fwd2 *Forwarder, face2 int, prefix string) {
    // face1 --> link1 --> face2 (input queue)
    fwd1.OutputFaceQueues[face1] = fwd1.OutputFaceQueues[1] // all queues dump to the default output queue
    if _, ok := fwd2.InputFaceQueues[face2]; !ok {
        fwd2.InputFaceQueues[face2] = queue{make(chan StagedMessage, 100), 10};
    }
    if _, ok := fwd1.FaceLinks[face1]; !ok {
        link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
        fwd1.FaceLinks[face1] = link;
    }
    fwd1.FaceLinkQueues[face1] = fwd2.InputFaceQueues[face2];

    // face2 --> link2 --> face1 (input queue)
    fwd2.OutputFaceQueues[face2] = fwd2.OutputFaceQueues[1] // all queues dump to the default output queue
    if _, ok := fwd1.InputFaceQueues[face1]; !ok {
        fwd1.InputFaceQueues[face1] = queue{make(chan StagedMessage, 100), 10};
    }
    if _, ok := fwd2.FaceLinks[face1]; !ok {
        link := link{make(chan StagedMessage, 10), 10, 0.0, 1000}
        fwd2.FaceLinks[face2] = link;
    }
    fwd2.FaceLinkQueues[face2] = fwd1.InputFaceQueues[face1];

    // repr, _ := json.Marshal(fwd2.InputFaceQueues);
    // fmt.Println(string(repr));

    // face connection
    fwd1.FaceToFace[face1] = face2;
    fwd2.FaceToFace[face2] = face1;

    // prefix configuration
    fwd1.Fib.AddPrefix(prefix, face1);
}

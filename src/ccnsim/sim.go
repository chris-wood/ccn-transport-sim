package ccnsim

import "math/rand"

import "ccnsim/core"

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
// TODO: move FaceManager to a separate struct
// the face manager will keep all faces, queues, and link references

func (s Simulator) Consumer_Create(id string) (*core.Consumer) {
    outputFaceMap := make(map[int]core.Queue);
    inputFaceMap := make(map[int]core.Queue);
    faceLinkMap := make(map[int]core.Link);
    faceLinkMapQueues := make(map[int]core.Queue);
    faceToFace := make(map[int]int);
    ofifo := core.Queue{make(chan core.StagedMessage, 100), 10};
    ififo := core.Queue{make(chan core.StagedMessage, 100), 10};
//    link := core.Link{make(chan core.StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan core.QueuedMessage, 10)

    defaultNodeFace := 1;
    outputFaceMap[defaultNodeFace] = ofifo;
    inputFaceMap[defaultNodeFace] = ififo;
//    faceLinkMap[defaultNodeFace] = link;

    fwd := &core.Forwarder{id, []int{defaultNodeFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &core.FibTable{Table: make(map[string]core.FibTableEntry)}, &core.ContentStore{},
        &core.PitTable{Table: make(map[string]core.PitEntry)}};
    consumer := &core.Consumer{fwd, 0, defaultNodeFace};
    return consumer;
}

// TODO: how to make Queue creation more flexible?

func (s Simulator) Producer_Create(id string) (*core.Producer) {
    outputFaceMap := make(map[int]core.Queue);
    inputFaceMap := make(map[int]core.Queue);
    faceLinkMap := make(map[int]core.Link);
    faceLinkMapQueues := make(map[int]core.Queue);
    faceToFace := make(map[int]int);
    ififo := core.Queue{make(chan core.StagedMessage, 100), 10};
    ofifo := core.Queue{make(chan core.StagedMessage, 100), 10};
    link := core.Link{make(chan core.StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan core.QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &core.Forwarder{id, []int{defaultFace}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &core.FibTable{Table: make(map[string]core.FibTableEntry)}, &core.ContentStore{},
        &core.PitTable{Table: make(map[string]core.PitEntry)}};
    producer := &core.Producer{fwd, 0};
    return producer;
}

func (s Simulator) Router_Create(id string) (*core.Router) {
    outputFaceMap := make(map[int]core.Queue);
    inputFaceMap := make(map[int]core.Queue);
    faceLinkMap := make(map[int]core.Link);
    faceLinkMapQueues := make(map[int]core.Queue);
    faceToFace := make(map[int]int);
    ofifo := core.Queue{make(chan core.StagedMessage, 100), 10};
    ififo := core.Queue{make(chan core.StagedMessage, 100), 10};
    link := core.Link{make(chan core.StagedMessage, 10), 10, 0.0, 1000}
    processingPackets := make(chan core.QueuedMessage, 10)

    defaultFace := 1;
    outputFaceMap[defaultFace] = ofifo;
    inputFaceMap[defaultFace] = ififo;
    faceLinkMap[defaultFace] = link;

    fwd := &core.Forwarder{id, []int{defaultFace, 2}, outputFaceMap, inputFaceMap, faceLinkMap,
        faceLinkMapQueues, faceToFace, processingPackets,
        &core.FibTable{Table: make(map[string]core.FibTableEntry)}, &core.ContentStore{make(map[string]core.Data)},
        &core.PitTable{make(map[string]core.PitEntry)}};
    router := &core.Router{fwd};
    return router;
}

func (s Simulator) Connect(fwd1 *core.Forwarder, face1 int, fwd2 *core.Forwarder, face2 int, prefix string, link core.Link) {
    // face1 --> link1 --> face2 (input Queue)
    fwd1.OutputFaceQueues[face1] = fwd1.OutputFaceQueues[1] // all Queues dump to the default output queue
    if _, ok := fwd2.InputFaceQueues[face2]; !ok {
        fwd2.InputFaceQueues[face2] = core.Queue{make(chan core.StagedMessage, 100), 10};
    }
    if _, ok := fwd1.FaceLinks[face1]; !ok {
        //link := core.Link{make(chan core.StagedMessage, 10), 10, 0.0, 1000}
        fwd1.FaceLinks[face1] = link;
    }
    fwd1.FaceLinkQueues[face1] = fwd2.InputFaceQueues[face2];

    // face2 --> link2 --> face1 (input Queue)
    fwd2.OutputFaceQueues[face2] = fwd2.OutputFaceQueues[1] // all Queues dump to the default output queue
    if _, ok := fwd1.InputFaceQueues[face1]; !ok {
        fwd1.InputFaceQueues[face1] = core.Queue{make(chan core.StagedMessage, 100), 10};
    }
    if _, ok := fwd2.FaceLinks[face1]; !ok {
        //link := core.Link{make(chan core.StagedMessage, 10), 10, 0.0, 1000}
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

    return
}

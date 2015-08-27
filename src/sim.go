package main

import "fmt"
import "os"
import "strconv"
import "container/list"

type message struct {
    name string
    payload []uint8
}

type interest struct {
    message
    hashRestriction []uint8
    keyIdRestriction []uint8
}

type data struct {
    message
}

type manifest struct {
    message 
}

type fibentry struct {
    prefix string
    interfaces []int
}

type fibtable struct {
    table map[string]fibentry
}

type pitentry struct {
    name string
    records []interest
}

type pittable struct {
    table map[string]pitentry
}

type cache struct {
    cs map[string]data
}

type forwarder struct {
    id string
    faces []int
    fib fibtable
    cs cache
    pit pittable
}

type consumer struct {
    forwarder
}

func (c consumer) SendInterest(msg interest) {
    fmt.Println("sending interest");
}

func (c consumer) ReceiveInterest(msg interest) {
    fmt.Println("received interest");
}

func (c consumer) ReceiveData(msg data) {
    fmt.Println("received data");
}

func (c consumer) ReceiveManifest(msg manifest) {
    fmt.Println("received manifest");
}

type producer struct {
    forwarder
}

type router struct {
    forwarder
}

type Event struct {
    desc string
    val int
}

func main() {
    fmt.Println("ccn-transport-sim v0.0.0")

    args := os.Args[1:]

    simulationTime, err := strconv.Atoi(args[0])
    if err != nil {
        fmt.Printf("Error: invalid simulation time %s\n", args[0]);
        os.Exit(-1);
    }

    events := list.New()
    deferredEvents := list.New()

    // test events
    eventA := new(Event);
    eventA.desc = "test";
    eventA.val = 5;
    events.PushBack(eventA);

    // network elements
    fwd1 := forwarder{"fwd1", []int{1, 2}, fibtable{}, cache{}, pittable{}};
    consumer := consumer{fwd1};
    consumer.SendInterest(interest{message{name: "lci:/foo/bar", payload: []uint8{0x00}}});

//   id string
//    faces []int
//   fib fibtable
//    cs cache
//    pit pittable
//
    for i := 1; i <= simulationTime; i++ {
        fmt.Printf("Time = %d...\n", i);

        for e := events.Front(); e != nil; e = e.Next() {
            events.Remove(e);
            event := e.Value.(*Event);
            fmt.Printf("Time %d Event = %d\n", i, event.val);

            if event.val > 0 {
                eventB := new(Event)
                eventB.desc = event.desc
                eventB.val = event.val - 1
                deferredEvents.PushBack(eventB);
            }
        }

        // ping pong event queues
        tmp := events;
        events = deferredEvents;
        deferredEvents = tmp;
    }
}

package main

import "fmt"
import "os"
import "strconv"
import "container/list"
import "ccnsim"
import "ccnsim/core"

func main() {
    fmt.Println("ccn-transport-sim v0.0.0");

    args := os.Args[1:]

    simulationTime, err := strconv.Atoi(args[0]);
    if err != nil {
        fmt.Printf("Error: invalid simulation time %s\n", args[0]);
        os.Exit(-1);
    }

    events := list.New()
    deferredEvents := list.New()

    // test events
    desc := "Send interest command";
    timeout := 5; // every 5 ticks
    eventA := core.Event{desc, timeout}
    events.PushBack(eventA);

    // network elements
    sim := ccnsim.Simulator{};
    consumer := sim.Consumer_Create("consumer1");
    producer := sim.Producer_Create("producer1");
    router1 := sim.Router_Create("router1");
    router2 := sim.Router_Create("router2");

    nodes := make([]ccnsim.Runnable, 4);
    nodes[0] = consumer;
    nodes[1] = router1;
    nodes[2] = router2;
    nodes[3] = producer;

    // Make some connections
    // TODO: link and route
    sim.Connect(consumer.Fwd, 1, router1.Fwd, 1, "/foo");
    sim.Connect(router1.Fwd, 2, router2.Fwd, 1, "/foo");
    sim.Connect(router2.Fwd, 2, producer.Fwd, 1, "/foo");

    for i := 1; i <= simulationTime; i++ {

        // event processing pipeline
        for e := events.Front(); e != nil; e = e.Next() {
            events.Remove(e);
            event := e.Value.(core.Event);
            if event.Val > 0 {
                eventB := core.Event{event.Desc, event.Val - 1}
                deferredEvents.PushBack(eventB);
            } else {
                // 1. send an interest
                msg := core.Interest_CreateSimple("/foo/bar");
                consumer.SendInterest(msg);

                // 2. create timeout to send another one
                eventB := core.Event{desc, timeout}
                events.PushBack(eventB)
            }
        }

        // move each node along
        for e := 0; e < len(nodes); e++ {
            node := nodes[e];
            node.Tick(i);
        }

        // ping pong event queues
        tmp := events;
        events = deferredEvents;
        deferredEvents = tmp;
    }
}

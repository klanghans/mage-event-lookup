package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/bmatcuk/doublestar"
)

type Event struct {
	Parent    string `json:"namespace"`
	Event     string `json:"event_name"`
	File      string `json:"file"`
	CodePool  string `json:"code_pool"`
	Observers []*Observer
}

type Observer struct {
	ObserverClass    string `json:"class"`
	ObserverMethod   string `json:"method"`
	ObserverNameHash string `json:"observer_name_hash"`
}

type EventCollection struct {
	allowedXmlNameSpace map[string]bool
	List                []*Event
}

var searchEvent, workingDir string

func init() {
	flag.StringVar(&searchEvent, "event", "", "event to search for")
	flag.StringVar(&workingDir, "dir", "", "directory to search in")
	flag.Parse()
}

func NewEventCollection() *EventCollection {
	ns := make(map[string]bool)

	return &EventCollection{
		allowedXmlNameSpace: ns,
	}
}

func main() {
	start := time.Now()

	if workingDir == "" {
		os.Stdout.Write([]byte("No working dir provided!"))
		os.Exit(128)
	}

	glob, err := doublestar.Glob(filepath.Clean(workingDir + "/**/config.xml"))
	if err != nil {
		panic(err)
	}

	eventCollection := NewEventCollection()

	for _, v := range glob {
		f, err := os.Open(v)
		if err != nil {
			panic(err)
		}

		doc, err := xmlquery.Parse(f)
		events := xmlquery.Find(doc, "//events")

		for _, eventList := range events {
			eventCollection.extractEvents(eventList, f)
		}
		f.Close()
	}

	if searchEvent == "" {
		eventCollection.print()
	} else {
		eventCollection.filterByEvent(searchEvent)
	}

	elapsed := time.Since(start)
	os.Stdout.Write([]byte(fmt.Sprintf("\nelapsed time: %s\n", elapsed)))
}

func (events *EventCollection) extractEvents(eventList *xmlquery.Node, f *os.File) {
	var observerEntry *xmlquery.Node
	nameSpace := getEventNameSpace(eventList)

	if nameSpace != "global" && nameSpace != "adminhtml" && nameSpace != "frontend" {
		return
	}

	event := eventList.FirstChild.NextSibling

	if event == nil {
		return
	}

EventLoop:
	for {
		e := &Event{}
		e.Parent = nameSpace
		e.File = f.Name()
		e.Event = event.Data
		e.CodePool = codePool(f)

		// disregard CommentNodes, commented XML nodes
		if event.Type == xmlquery.CommentNode {
			return
		}

		observers := event.SelectElement("observers")
		if observers == nil {
			continue
		}
		observerEntry = observers.FirstChild.NextSibling

	ObserverLoop:
		for {
			// get observers
			o := &Observer{}
			if observers != nil {
				o.ObserverNameHash = observerEntry.Data
			}

			class := observerEntry.SelectElement("class")
			if class != nil {
				o.ObserverClass = class.FirstChild.Data
			}

			method := observerEntry.SelectElement("method")
			if method != nil {
				o.ObserverMethod = method.FirstChild.Data
			}

			e.Observers = append(e.Observers, o)

			events.List = append(events.List, e)

			if observerEntry.NextSibling == nil {
				break ObserverLoop
			} else {
				observerEntry = observerEntry.NextSibling
			}
		}

		if event.NextSibling == nil {
			break EventLoop
		} else {
			event = event.NextSibling
		}

	}

}

func (events *EventCollection) print() {
	j, err := json.MarshalIndent(events.List, "", "\t")
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(j)
}

func (events *EventCollection) filterByEvent(event string) {
	filtered := make([]*Event, 0)
	for _, e := range events.List {
		if strings.Contains(e.Event, event) {
			filtered = append(filtered, e)
		}
	}
	j, err := json.MarshalIndent(filtered, "", "\t")
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(j)

}

func codePool(f *os.File) (pool string) {
	parts := strings.Split(f.Name(), string(filepath.Separator))

PartsLoop:
	for i := 0; i < len(parts); i++ {
		if parts[i-1] == "code" {
			pool = parts[i]
			break PartsLoop
		}
	}

	return pool
}

func getEventNameSpace(node *xmlquery.Node) string {
	if p := node.Parent; p != nil {
		return p.Data
	}

	return ""
}

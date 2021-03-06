package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
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

var searchEvent, workingDir, mode string
var benchmark bool

func init() {
	flag.StringVar(&searchEvent, "event", "", "event to search for")
	flag.StringVar(&workingDir, "dir", "", "directory to search in")
	flag.StringVar(&mode, "mode", "", "modus to display: list (default), graph")
	flag.BoolVar(&benchmark, "b", false, "display benchmark time")
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
		_, wErr := os.Stdout.Write([]byte("No working dir provided!\n"))
		if wErr != nil {
			panic(wErr)
		}

		os.Exit(128)
	}

	if strings.HasPrefix(workingDir, "~/") {
		workingDir = expandTilde(workingDir)
	}

	glob, err := doublestar.Glob(filepath.Clean(workingDir + "/**/config.xml"))
	if err != nil {
		panic(err)
	}

	eventCollection := NewEventCollection()

	switch mode {
	case "graph":
		buildCollection(glob, eventCollection)
		eventCollection.enrichWithDispatch()
	case "list":
	default:
		buildCollection(glob, eventCollection)
		if searchEvent == "" {
			eventCollection.print()
		} else {
			eventCollection.filterByEvent(searchEvent)
		}
	}

	elapsed := time.Since(start)

	if benchmark {
		_, err = os.Stdout.Write([]byte(fmt.Sprintf("\nelapsed time: %s\n", elapsed)))
		if err != nil {
			panic(err)
		}
	}
}

func buildCollection(glob []string, eventCollection *EventCollection) {
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
		cErr := f.Close()
		if cErr != nil {
			panic(cErr)
		}
	}
}

func expandTilde(path string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	return filepath.Join(dir, path[2:])
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

func (events *EventCollection) enrichWithDispatch() {
	// find all dispatchEvent in code under app/code/**/
}

func (events *EventCollection) print() {
	j, err := json.MarshalIndent(events.List, "", "\t")
	if err != nil {
		panic(err)
	}
	_, err = os.Stdout.Write(j)
	if err != nil {
		panic(err)
	}
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
	_, err = os.Stdout.Write(j)
	if err != nil {
		panic(err)
	}
}

func codePool(f *os.File) (pool string) {
	parts := strings.Split(f.Name(), string(filepath.Separator))

PartsLoop:
	for i := 1; i < len(parts); i++ {
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

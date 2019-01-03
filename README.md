# mage-event-lookup
### Event Lookup tool for Magento 1.x
A simple tool, that does one thing.
It let's you search all the events in magento config.xml files that contain your search word.

Even though my IDE lets me search through the codebase, the information it contains is always hidden from plain sight.

Use cases:
- search for a partial event name and get all registered Observers returned, sorted by Magento code pool. This lets you figure out: 
  - What event names exists at all
  - What observers are subscribed on it
- search for a specific event name and get all registered Observers returned, sorted by Magento code pool

The json formated response let you parse the result in a programmatic way.

####json formated results:
```
[

{
                "namespace": "global",
                "event_name": "sales_order_save_commit_after",
                "file": "test/app/code/community/CommunityVendorName2/ModuleName2/etc/config.xml",
                "code_pool": "community",
                "Observers": [
                        {
                                "class": "namespace_search_indexer/observer",
                                "method": "someObserverMethodTwo",
                                "observer_name_hash": "index_products_from_order"
                        }
                ]
        }
...
]

```
### Usage
#### build
```
go build -o mage_event_lookup main.go
```

#### search events
```
./mage_event_lookup --dir ./test --event=catalog
```

#### tilde expansion
```
./mage_event_lookup --dir ~/your-magento-path --event=catalog
```



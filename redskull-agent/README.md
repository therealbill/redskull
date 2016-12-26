# Package Documentation

To import:

```go
import "github.com/therealbill/redskull/rpcclient"
```


# Sample Usage

Say you want a utility which will be run on startup of a sentinel server to
"register" the local sentinel instance. This way new sentinels are already in
your constellation. In order to keep this simple you'e set up a TCP load
balance so your tools can simply call one DNS address and be connected to your
Redskull cluster.

The relevant portion of the tool could look like the following:

```go
//set a timeout of 5 seconds
timeout := time.Second * 5
client, err:= NewClient("myredskull.host.name:8001" , timeout ) 
if err != nil {
	log.Fatal("Unable to connect to Redskull cluster")
}
ok, err := client.AddSentinel("my.local.sentinel.ip:port")
if err != nil {
	log.Fatalf("Unable to add sentinel. Error was '%s'",err)
}

if !ok {
	log.Fatal("Redskull returned no error, but didn't register the new sentinel. Look into it's logs.")
}
//whatever you want to do from here.
```

As you can see we connect, check for valid connection, then attempt to add our
local sentinel.  The "tricky" bits would be determining the IP to register.
Ideally your sentinel config file would have it in the `bind` statement but
that isn't always the address you need to use (as might be the case in a
container behind NAT for example.

# Types

## Client
```go
type Client struct { // contains filtered or unexported fields }
```

## NewPodRequest

NewPodRequest is a struct used for passing in the pod information from the
rpc client to the server, and is not generally used by the client code.

```go
type NewPodRequest struct {
    Name   string
    IP     string
    Port   int
    Quorum int
    Auth   string
}
```

# Functions

## NewClient
NewClient returns a client connection. It takes a connection string in the form
of "host:port", and a time.Duration for the timeout.

```go
func NewClient(dsn string, timeout time.Duration) (*Client, error)
```

##AddPod
AddPod(NewPodRequest) will take the information in the PodRequest and instruct
Redskull to add it to it's monitor list. It is used by rpc clients to marshal
up the necessary parameters.

```go
func (c *Client) AddPod(name, ip string, port, quorum int, auth string) (actions.RedisPod, error)
```

##AddSentinel
AddSentinel(address) will instuct Redskull to add the sentinel at the given
address.

```go
func (c *Client) AddSentinel(address string) (bool, error)
```

##BalancePod
BalancePod will attempt to rebalance the pod's sentinels.

```go
func (c *Client) BalancePod(podname string) error
```

##GetPod
GetPod(podname) will return an `actions.RedisPod` type for the given pod,
if found, or an empty one and an error if not.
```go
func (c *Client) GetPod(podname string) (actions.RedisPod, error)
```


##GetSentinelsForPod
GetSentinelsForPod(podname) returns the number and list of sentinels for
the given podname.

```go
func (c *Client) GetSentinelsForPod(address string) (int, []string, error)
```

##RemovePod
RemovePod(podname) removes the pod from Redskull and associated
sentinels.

```go
func (c *Client) RemovePod(podname string) error
```


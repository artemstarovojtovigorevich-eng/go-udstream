# UDStream - One‑Way UDP Protocol for OPC UA‑Like Data

UDStream is a lightweight Go library that lets you send structured data (like OPC UA node values) over UDP in one direction:  
**client → server**, without server responses.  
It’s similar to gRPC + Protobuf, but without RPC semantics and built on top of UDP.

---

## 1. Quick Overview

- Transports `Message`/`DeltaBatch`/`FullSnapshot` as Protobuf over UDP.  
- Client sends data (e.g. OPC UA‑node updates).  
- Server receives and updates an internal state model (e.g. OPC UA‑mirror).  
- No ACKs, no TCP, no bidirectional RPC — just fire‑and‑forget UDP‑style streaming.

---

## 2. Installing UDStream

Create your module (if not yet):

```bash
cd ~/Desktop/udproto
go mod init github.com/yourusername/go-udstream
```

Make sure your `go.mod` contains:

```go
require google.golang.org/protobuf v1.36.11
```

Then run:

```bash
go mod tidy
```

---

## 3. Protocol Messages

All messages are defined in `proto/udstream.proto`:

```proto
message Message {
  uint64 timestamp_ns = 1;
  string node_id = 2;          # OPC UA‑like NodeID
  Value  value = 3;
  uint32 source_id = 4;        # unique client ID
}

message DeltaBatch {
  uint32 seq = 1;              # sequence number
  repeated Message messages = 2; # changed nodes
}

message FullSnapshot {
  uint64 timestamp = 1;
  uint32 source_id = 2;
  repeated Message nodes = 3;   # all nodes
}

message Packet {
  oneof payload {
    DeltaBatch delta = 1;
    FullSnapshot full = 2;
  }
}
```

Generated Go code lives in `proto/pb/udstream.pb.go`.

---

## 4. Client Usage (UDP Sender)

Use `udstream.Client` to send `Delta` or `Full` packets to a UDP server.

### 4.1 Create the client

```go
client, err := udstream.NewClient("127.0.0.1:33333", 1)
if err != nil {
  panic(err)
}
defer client.Close()
```

- `"127.0.0.1:33333"` — address of your UDP server.  
- `1` — `source_id` (client ID).

### 4.2 Send a delta update (e.g. OPC node changes)

```go
msg := &pb.Message{
  TimestampNs: uint64(time.Now().UnixNano()),
  NodeId:      "ns=2;s=TemperatureSensor",
  Value: &pb.Value{
    Value: &pb.Value_DoubleValue{DoubleValue: 23.5},
  },
  SourceId: 1,
}

err = client.SendDelta(ctx, 42, []*pb.Message{msg})
if err != nil {
  // handle network error (e.g. retry or log)
}
```

- `ctx` — optional context for cancellation.  
- `42` — `seq` number.  
- `client.SendFull(...)` works similarly for full snapshots.

---

## 5. Server Usage (UDP Listener)

The server exposes a `udstream.Server` that calls your callbacks on incoming packets.

### 5.1 Implement the handler

```go
type OPCMirrorHandler struct {
  // your OPC UA‑like state store (map, DB, etc.)
}

func (h *OPCMirrorHandler) OnDelta(delta *pb.DeltaBatch) {
  for _, msg := range delta.Messages {
    // update your OPC UA‑m
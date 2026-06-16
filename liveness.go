package contracts

import "time"

// Liveness sinks transport-level keepalives (e.g. a gateway heartbeat ACK) so the
// host can judge whether its connection is still live. A CommandSource that has a
// keepalive signal may accept one via an optional SetLiveness(Liveness) method;
// the host wires its own health into it when present.
type Liveness interface {
	HeartbeatAck(t time.Time)
}

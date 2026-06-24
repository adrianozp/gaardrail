// Package staticqueue adapts a non-switchable queue (kafka/sqs) to the
// switchqueue.Queue capability: it reports the configured type, offers no
// alternatives, and rejects any switch. This keeps the /queue/type endpoint
// valid regardless of the configured protocol.
package staticqueue

import "fmt"

type Queue struct {
	protocol string
}

func New(protocol string) Queue {
	return Queue{protocol: protocol}
}

func (q Queue) Type() string { return q.protocol }

func (q Queue) Available() []string { return []string{} }

func (q Queue) SetType(string) error {
	return fmt.Errorf("queue: %q is not switchable at runtime", q.protocol)
}

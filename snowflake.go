// Package snowflake provides a very simple Twitter snowflake generator and parser.
package snowflake

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	nodeBits        = 10
	stepBits        = 12
	nodeMax         = -1 ^ (-1 << nodeBits)
	stepMask  int64 = -1 ^ (-1 << stepBits)
	timeShift uint8 = nodeBits + stepBits
	nodeShift uint8 = stepBits
)

// Epoch is set to the twitter snowflake epoch of 2006-03-21:20:50:14 GMT
// You may customize this to set a different epoch for your application.
var Epoch int64 = 1288834974657

// A Node struct holds the basic information needed for a snowflake generator
// node
type Node struct {
	sync.Mutex
	time int64
	node int64
	step int64
}

// An ID is a custom type used for a snowflake ID.  This is used so we can
// attach methods onto the ID.
type ID int64

// NewNode returns a new snowflake node that can be used to generate snowflake
// IDs
func NewNode(node int64) (*Node, error) {

	if node < 0 || node > nodeMax {
		return nil, errors.New("Node number must be between 0 and 1023")
	}

	return &Node{
		time: 0,
		node: node,
		step: 0,
	}, nil
}

// NewNodeByHostname is a convenience method which creates a new Node based
// off a hash of the machine's hostname.
func NewNodeByHostname() (*Node, error) {
	name, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	hash := md5.Sum([]byte(name))
	id := binary.BigEndian.Uint64(hash[:]) & 0x3FF // mask to first 10 bits, max of 1023

	return NewNode(int64(id))
}

// Generate creates and returns a unique snowflake ID
func (n *Node) Generate() ID {

	n.Lock()

	now := time.Now().UnixNano() / 1000000

	if n.time == now {
		n.step = (n.step + 1) & stepMask

		if n.step == 0 {
			for now <= n.time {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		n.step = 0
	}

	n.time = now

	r := ID((now-Epoch)<<timeShift |
		(n.node << nodeShift) |
		(n.step),
	)

	n.Unlock()
	return r
}

// Int64 returns an int64 of the snowflake ID
func (f ID) Int64() int64 {
	return int64(f)
}

// String returns a string of the snowflake ID
func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

// Base2 returns a string base2 of the snowflake ID
func (f ID) Base2() string {
	return strconv.FormatInt(int64(f), 2)
}

// Base36 returns a base36 string of the snowflake ID
func (f ID) Base36() string {
	return strconv.FormatInt(int64(f), 36)
}

// Base64 returns a base64 string of the snowflake ID
func (f ID) Base64() string {
	return base64.StdEncoding.EncodeToString(f.Bytes())
}

// Bytes returns a byte array of the snowflake ID
func (f ID) Bytes() []byte {
	return []byte(f.String())
}

// Time returns an int64 unix timestamp of the snowflake ID time
func (f ID) Time() int64 {
	return (int64(f) >> 22) + Epoch
}

// Node returns an int64 of the snowflake ID node number
func (f ID) Node() int64 {
	return int64(f) & 0x00000000003FF000 >> nodeShift
}

// Step returns an int64 of the snowflake step (or sequence) number
func (f ID) Step() int64 {
	return int64(f) & 0x0000000000000FFF
}

// MarshalJSON returns a json byte array string of the snowflake ID.
func (f ID) MarshalJSON() ([]byte, error) {
	buff := make([]byte, 0, 22)
	buff = append(buff, '"')
	buff = strconv.AppendInt(buff, int64(f), 10)
	buff = append(buff, '"')
	return buff, nil
}

// UnmarshalJSON converts a json byte array of a snowflake ID into an ID type.
func (f *ID) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(string(b[1:len(b)-1]), 10, 64)
	if err != nil {
		return err
	}

	*f = ID(i)
	return nil
}

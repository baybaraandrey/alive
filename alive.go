// Package alive is a simple hosts availability watcher library that used
// icmp protocol

package alive

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"gopkg.in/yaml.v3"
)

const (
	ProtocolICMP = 1
)

var (
	ipv4Proto = map[string]string{"icmp": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"icmp": "ip6:ipv6-icmp", "udp": "udp6"}
)

// Default to listen on all IPv4 interfaces
var defaulListenAddr = "0.0.0.0"

// New returns a new Watcher struct pointer
func New(addr string, logger *log.Logger) *Watcher {
	rand.Seed(time.Now().UnixNano())
	return &Watcher{
		Interval: time.Second,
		Timeout:  time.Duration(math.MaxInt64),
		Size:     0,
		Source:   defaulListenAddr,

		addr:     addr,
		done:     make(chan interface{}),
		id:       rand.Intn(math.MaxInt64),
		ipaddr:   nil,
		ipv4:     false,
		network:  "ip",
		protocol: "udp",
		TTL:      64,
		logger:   logger,
	}
}

// NewWatcher returns a new Watcher and resolves the address
func NewWatcher(addr string, logger *log.Logger) (*Watcher, error) {
	w := New(addr, logger)
	return w, w.Resolve()
}

// Watcher ...
type Watcher struct {
	Interval     time.Duration
	Timeout      time.Duration
	ReadDeadline time.Duration

	// Size of packet being sent
	Size uint

	// Source is the source IP address
	Source string

	// Channel and mutex used to communicate when the Watcher should stop between goroutines.
	done chan interface{}
	lock sync.Mutex

	ipaddr *net.IPAddr
	addr   string

	ipv4     bool
	id       int
	sequence int

	// network is one of "ip", "ip4", or "ip6".
	network string
	// protocol is "icmp" or "udp".
	protocol string

	logger *log.Logger

	TTL int
}

// SetSource sets interface to listen packets on
func (w *Watcher) SetSource(source string) {
	w.Source = source
}

// SetTimeout sets global timeout for listening
func (w *Watcher) SetTimeout(timeout time.Duration) {
	w.Timeout = timeout
}

// SetReadDeadline sets deadline to read reply packet
func (w *Watcher) SetReadDeadline(deadline time.Duration) {
	w.ReadDeadline = deadline
}

// SetInterval sets interval to send packets
func (w *Watcher) SetInterval(interval time.Duration) {
	w.Interval = interval
}

// SetTTL sets time to live on packets
func (w *Watcher) SetTTL(ttl uint) {
	w.TTL = int(ttl)
}

// SetSize sets size of the packets body in bytes
func (w *Watcher) SetSize(size uint) {
	w.Size = size
}

// Addr returns the string ip address of the target host.
func (w *Watcher) Addr() string {
	return w.addr
}

// OnTimeout calls when global timeout occur
func (w *Watcher) OnTimeout() {
	fmt.Printf("%s: timeout\n", w.addr)
}

// OnRecv calls when packet receive
func (w *Watcher) OnRecv(ps *PacketStat) {
	fmt.Printf("%s | %s: icmp_seq=%d ttl=%d time=%v\n",
		w.addr, w.ipaddr.String(), w.sequence, w.TTL, ps.Duration)
}

// OnError calls when error occurs related to listening/reading packets
func (w *Watcher) OnError(err error) {
	fmt.Printf("%s | %s: error: %s : icmp_seq=%d ttl=%d\n",
		w.addr, w.ipaddr.String(), err, w.sequence, w.TTL)
}

// Resolve does the DNS lookup for the Pinger address and sets IP protocol.
func (w *Watcher) Resolve() error {
	if len(w.addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	ipaddr, err := net.ResolveIPAddr(w.network, w.addr)
	if err != nil {
		return err
	}

	w.ipv4 = isIPv4(ipaddr.IP)
	w.ipaddr = ipaddr

	return nil
}

func isIPv4(ip net.IP) bool {
	return strings.Contains(ip.String(), ".")
}

// Run runs the watcher.
func (w *Watcher) Run() error {
	var err error
	var conn *icmp.PacketConn

	if w.ipaddr == nil {
		err = w.Resolve()

	}
	if err != nil {
		return err
	}

	if conn, err = w.listen(); err != nil {
		return err
	}

	return w.run(conn)
}

func (w *Watcher) listen() (*icmp.PacketConn, error) {
	var (
		conn *icmp.PacketConn
		err  error
	)

	if w.ipv4 {
		conn, err = icmp.ListenPacket(ipv4Proto[w.protocol], w.Source)
	} else {
		conn, err = icmp.ListenPacket(ipv6Proto[w.protocol], w.Source)
	}

	if err != nil {
		w.Stop()
		return nil, err
	}

	return conn, nil

}

func (w *Watcher) run(conn *icmp.PacketConn) error {
	timeout := time.NewTicker(w.Timeout)
	interval := time.NewTicker(w.Interval)
	recv := make(chan *PacketStat, 5)

	defer func() {
		timeout.Stop()
		interval.Stop()
		close(recv)
	}()

	for {
		select {
		case <-w.done:
			return nil
		case <-timeout.C:
			w.OnTimeout()
		case <-interval.C:
			err := w.sendRecvICMP(conn, recv)
			if err != nil {
				w.OnError(err)
			}
		case msg := <-recv:
			w.OnRecv(msg)
		}
	}
	return nil
}

// PacketStat represents packet statistics
type PacketStat struct {
	Message  *icmp.Message
	Duration time.Duration
}

func (w *Watcher) sendRecvICMP(conn *icmp.PacketConn, recv chan *PacketStat) error {
	var dst net.Addr = w.ipaddr
	if w.protocol == "udp" {
		dst = &net.UDPAddr{IP: w.ipaddr.IP, Zone: w.ipaddr.Zone}
	}

	// Make a new ICMP message
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   w.id,
			Seq:  w.sequence,
			Data: bytes.Repeat([]byte("a"), int(w.Size)),
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	// Send it
	start := time.Now()
	n, err := conn.WriteTo(b, dst)
	if err != nil {
		return err
	} else if n != len(b) {
		return fmt.Errorf("got %v; want %v", n, len(b))
	}

	// Wait for a reply
	reply := make([]byte, 1500)
	err = conn.SetReadDeadline(time.Now().Add(w.ReadDeadline))
	if err != nil {
		return err
	}
	n, peer, err := conn.ReadFrom(reply)
	if err != nil {
		return err
	}
	duration := time.Since(start)

	// Pack it up boys, we're done here
	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		recv <- &PacketStat{rm, duration}
	default:
		return fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}

	return nil
}

// Stop stops watcher
func (w *Watcher) Stop() {
	w.lock.Lock()
	defer w.lock.Unlock()

	open := true
	select {
	case _, open = <-w.done:
	default:
	}

	if open {
		close(w.done)
	}
}

// SetPrivileged sets the type of ping pinger will send.
// false means pinger will send an "unprivileged" UDP ping.
// true means pinger will send a "privileged" raw ICMP ping.
// NOTE: setting to true requires that it be run with super-user privileges.
func (w *Watcher) SetPrivileged(privileged bool) {
	if privileged {
		w.protocol = "icmp"
	} else {
		w.protocol = "udp"
	}
}

// HostConfig structure to configurate host watcher
type HostConfig struct {
	Addr        string `yaml:"addr"`
	Interval    string `yaml:"interval"`
	ReadTimeout string `yaml:"read-timeout"`
	Size        uint   `yaml:"packet-size"`
	TTL         uint   `yaml:"ttl"`
}

// Config structure that holds hosts configs
type Config struct {
	Hosts []HostConfig `yaml:"hosts"`
}

// ReadConfig reads and returns parsed config for hosts
func ReadConfig(path string) (*Config, error) {
	var cfg = &Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

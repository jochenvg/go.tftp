package tftp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// opcode is a TFTP packet opcode
type opcode uint16

//go:generate stringer -type=opcode

// opcode constants
const (
	_     opcode = iota
	RRQ          // RFC 1350 The TFTP Protocol (Revision 2)
	WRQ          // RFC 1350 The TFTP Protocol (Revision 2)
	DATA         // RFC 1350 The TFTP Protocol (Revision 2)
	ACK          // RFC 1350 The TFTP Protocol (Revision 2)
	ERROR        // RFC 1350 The TFTP Protocol (Revision 2)
	OACK         // RFC 2347 TFTP option Extension
	maxopcode
)

// Mode is a TFTP transfer mode
type Mode uint8

//go:generate stringer -type=Mode

// Mode constants
const (
	_        Mode = iota
	Octet         // RFC 1350 The TFTP Protocol (Revision 2)
	Netascii      // RFC 1350 The TFTP Protocol (Revision 2)
	Mail          // RFC 1350 The TFTP Protocol (Revision 2)
	maxMode
)

// option is a TFTP option
type option uint8

//go:generate stringer -type=option

// option constants
const (
	_          option = iota
	blksize           // RFC 2348 TFTP Blocksize option
	timeout           // RFC 2349 TFTP Timeout Interval and Transfer Size Options
	tsize             // RFC 2349 TFTP Timeout Interval and Transfer Size Options
	multicast         // RFC 2090 TFTP Multicast option
	windowsize        // RFC 7440 TFTP Windowsize option
	maxOption
)

// block is a TFTP packet block number
type block uint16

// errorCode is the TFTP packet error code
type errorCode uint16

//go:generate stringer -type=errorCode

// errorCode constants
const (
	_                 errorCode = iota
	FileNotFound                // RFC 1350 The TFTP Protocol (Revision 2)
	AccessViolation             // RFC 1350 The TFTP Protocol (Revision 2)
	DiskFull                    // RFC 1350 The TFTP Protocol (Revision 2)
	IllegalOperation            // RFC 1350 The TFTP Protocol (Revision 2)
	UnknownTransferID           // RFC 1350 The TFTP Protocol (Revision 2)
	FileAlreadyExists           // RFC 1350 The TFTP Protocol (Revision 2)
	NoSuchUser                  // RFC 1350 The TFTP Protocol (Revision 2)
	maxErrorCode
)

// packet is a TFTP packet
type packet []byte

var separator = []byte{0}

// opcode gets the opcode
func (p packet) opcode() (o opcode) {
	if len(p) >= 2 {
		o = opcode(binary.BigEndian.Uint16(p[:2]))
	}
	return
}

// Filename gets the filename in a RRQ or WRQ
func (p packet) filename() (s string) {
	switch p.opcode() {
	case RRQ, WRQ:
		parts := bytes.SplitN(p[2:], separator, 2)
		if len(parts) >= 2 {
			s = string(parts[0])
		}
	}
	return
}

// Mode gets the mode
func (p packet) mode() (m Mode) {
	switch p.opcode() {
	case RRQ, WRQ:
		parts := bytes.SplitN(p[2:], separator, 3)
		if len(parts) >= 3 {
			switch strings.ToLower(string(parts[1])) {
			case "octet":
				m = Octet
			case "netascii":
				m = Netascii
			case "mail":
				m = Mail
			}

		}
	}
	return
}

// Options gets the options
func (p packet) options() (o map[option]int) {
	opcode := p.opcode()
	parts := bytes.Split(p[2:], separator)
	fmt.Println(parts)
	if len(parts) >= 2 {
		switch opcode {
		case RRQ, WRQ:
			parts = parts[2:]
		}
		switch opcode {
		case RRQ, WRQ, OACK:
			o = make(map[option]int)
			for len(parts) >= 2 {
				var option option
				var val int
				var err error
				name := strings.ToLower(string(parts[0]))
				value := string(parts[1])
				parts = parts[2:]
				switch name {
				case "blksize":
					if val, err = strconv.Atoi(value); err != nil {
						continue
					}
					option = blksize
				case "timeout":
					if val, err = strconv.Atoi(value); err != nil {
						continue
					}
					option = timeout
				case "tsize":
					if val, err = strconv.Atoi(value); err != nil {
						continue
					}
					option = tsize
				case "multicast":
					if len(value) != 0 {
						continue
					}
					val = 0
					option = multicast
				case "windowsize":
					if val, err = strconv.Atoi(value); err != nil {
						continue
					}
					option = windowsize
				default:
					continue
				}
				o[option] = val
			}
		}
	}
	return
}

// block gets the block number
func (p packet) block() (b block) {
	if len(p) >= 4 {
		opcode := p.opcode()
		switch opcode {
		case ACK, DATA:
			b = block(binary.BigEndian.Uint16(p[2:4]))
		}
	}
	return
}

// errorCode gets the error code
func (p packet) errorCode(e errorCode) {
	if len(p) >= 4 {
		switch p.opcode() {
		case ERROR:
			e = errorCode(binary.BigEndian.Uint16(p[2:4]))
		}
	}
	return
}

// Data gets the data
func (p packet) data() (d []byte) {
	if len(p) >= 4 {
		switch p.opcode() {
		case DATA:
			d = p[4:]
		}
	}
	return
}

// ErrorMessage gets the error message
func (p packet) errorMessage() (e string) {
	if len(p) >= 4 {
		p = p[4:]
		if i := bytes.IndexByte(p, 0); i != -1 {
			e = string(p[:i])
		}
	}
	return
}

func writeOptions(out io.Writer, options map[option]int) {
	for option, value := range options {
		fmt.Fprintf(out, "%s\x00", option.String())
		if option != multicast {
			fmt.Fprintf(out, "%d\x00", value)
		} else {
			fmt.Fprintf(out, "\x00")
		}
	}
}

func writeRequest(out io.Writer, opcode opcode, filename string, mode Mode, options map[option]int) {
	binary.Write(out, binary.BigEndian, uint16(opcode))
	fmt.Fprintf(out, "%s\x00", filename)
	fmt.Fprintf(out, "%s\x00", mode.String())
	writeOptions(out, options)
}

// newRRQPacket returns a packet containing a new RRQ packet
func newRRQPacket(filename string, mode Mode, options map[option]int) packet {
	out := &bytes.Buffer{}
	writeRequest(out, RRQ, filename, mode, options)
	return out.Bytes()
}

// newWRQPacket returns a packet containing a new RRQ packet
func newWRQPacket(filename string, mode Mode, options map[option]int) packet {
	out := &bytes.Buffer{}
	writeRequest(out, WRQ, filename, mode, options)
	return out.Bytes()
}

// newDATAPacket returns a packet containing a new DATA packet
func newDATAPacket(block block, data []byte) packet {
	out := &bytes.Buffer{}
	binary.Write(out, binary.BigEndian, uint16(DATA))
	binary.Write(out, binary.BigEndian, uint16(block))
	out.Write(data)
	return out.Bytes()
}

// newACKPacket returns a packet containing a new ACK packet
func newACKPacket(block block) packet {
	out := &bytes.Buffer{}
	binary.Write(out, binary.BigEndian, uint16(ACK))
	binary.Write(out, binary.BigEndian, uint16(block))
	return out.Bytes()
}

// newERRORPacket returns a packet containing a new ERROR packet
func newERRORPacket(errorcode errorCode, errormessage string) packet {
	out := &bytes.Buffer{}
	binary.Write(out, binary.BigEndian, uint16(ERROR))
	binary.Write(out, binary.BigEndian, uint16(errorcode))
	fmt.Fprintf(out, "%s\x00", errormessage)
	return out.Bytes()
}

// newOACKPacket returns a packet containing a new OACK packet
func newOACKPacket(options map[option]int) packet {
	out := &bytes.Buffer{}
	binary.Write(out, binary.BigEndian, uint16(OACK))
	writeOptions(out, options)
	return out.Bytes()
}

// ReadHandler is a handler function type for a read handler
type ReadHandler func(filename string, mode Mode) (io.ReadCloser, error)

// WriteHandler is a handler function type for a write handler
type WriteHandler func(filename string, mode Mode) (io.WriteCloser, error)

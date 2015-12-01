package tftp

import (
	"fmt"
	"testing"
)

var validPacketStrings = []string{
	"\x00\x01test\x00mail\x00",
	"\x00\x02test\x00netascii\x00",
	"\x00\x02test\x00octet\x00blksize\x001024\x00tsize\x000\x00timeout\x0010\x00multicast\x00\x00windowsize\x0016\x00",
	"\x00\x03\xbb\xaadata",
	"\x00\x04\xbb\xaa",
	"\x00\x05\xee\xccerror message\x00",
	"\x00\x06blksize\x001024\x00tsize\x000\x00timeout\x0010\x00multicast\x00\x00windowsize\x0016\x00",
}

type parts struct {
	opcode   opcode
	filename string
	mode     Mode
	block    block
}

var validParts = []parts{
	{RRQ, "test", Mail, 0},
	{WRQ, "test", Netascii, 0},
	{WRQ, "test", Octet, 0},
	{DATA, "", 0, 0xbbaa},
	{ACK, "", 0, 0xbbaa},
	{ERROR, "", 0, 0},
	{OACK, "", 0, 0},
}

func TestPacket(t *testing.T) {

	for i, s := range validPacketStrings {
		p := packet(s)
		if p.opcode() != validParts[i].opcode {
			fmt.Println(p.opcode().String())
			t.Fail()
		}
		if p.filename() != validParts[i].filename {
			fmt.Println(p.filename())
			t.Fail()
		}
		if p.mode() != validParts[i].mode {
			fmt.Println(p.mode().String())
			t.Fail()
		}
		if p.block() != validParts[i].block {
			t.Fail()
		}
		fmt.Println(p.options())
	}

}

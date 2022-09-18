package dns

import (
	"fmt"

	"github.com/brunexgeek/hugye/pkg/binary"
)

func RCodeToString(value int) string {
	switch value {
	case 0:
		return "NOERROR"
	case 1:
		return "FORMERR"
	case 2:
		return "SERVFAIL"
	case 3:
		return "NXDOMAIN"
	case 4:
		return "NOTIMP"
	case 5:
		return "REFUSED"
	case 6:
		return "YXDOMAIN"
	case 7:
		return "XRRSET"
	case 8:
		return "NOTAUTH"
	case 9:
		return "NOTZONE"
	default:
		return "?????"
	}
}

func TypeToString(value int) string {
	switch value {
	case 1:
		return "A" // RFC 1035[1]
	case 2:
		return "NS" // RFC 1035[1]
	case 5:
		return "CNAME" // RFC 1035[1]
	case 6:
		return "SOA" // RFC 1035[1] and RFC 2308[11]
	case 12:
		return "PTR" // RFC 1035[1]
	case 13:
		return "HINFO" // RFC 8482
	case 15:
		return "MX" // RFC 1035[1] and RFC 7505
	case 16:
		return "TXT" // RFC 1035[1]
	case 17:
		return "RP" // RFC 1183
	case 18:
		return "AFSDB" // RFC 1183
	case 24:
		return "SIG" // RFC 2535
	case 25:
		return "KEY" // RFC 2535[3] and RFC 2930[4]
	case 28:
		return "AAAA" // RFC 3596[2]
	case 29:
		return "LOC" // RFC 1876
	case 33:
		return "SRV" // RFC 2782
	case 35:
		return "NAPTR" // RFC 3403
	case 36:
		return "KX" // RFC 2230
	case 37:
		return "CERT" // RFC 4398
	case 39:
		return "DNAME" // RFC 6672
	case 42:
		return "APL" // RFC 3123
	case 43:
		return "DS" // RFC 4034
	case 44:
		return "SSHFP" // RFC 4255
	case 45:
		return "IPSECKEY" // RFC 4025
	case 46:
		return "RRSIG" // RFC 4034
	case 47:
		return "NSEC" // RFC 4034
	case 48:
		return "DNSKEY" // RFC 4034
	case 49:
		return "DHCID" // RFC 4701
	case 50:
		return "NSEC3" // RFC 5155
	case 51:
		return "NSEC3PARAM" // RFC 5155
	case 52:
		return "TLSA" // RFC 6698
	case 53:
		return "SMIMEA" // RFC 8162[9]
	case 55:
		return "HIP" // RFC 8005
	case 59:
		return "CDS" // RFC 7344
	case 60:
		return "CDNSKEY" // RFC 7344
	case 61:
		return "OPENPGPKEY" // RFC 7929
	case 62:
		return "CSYNC" // RFC 7477
	case 63:
		return "ZONEMD" // RFC 8976
	case 64:
		return "SVCB" // IETF Draft
	case 65:
		return "HTTPS" // IETF Draft
	case 108:
		return "EUI48" // RFC 7043
	case 109:
		return "EUI64" // RFC 7043
	case 249:
		return "TKEY" // RFC 2930
	case 250:
		return "TSIG" // RFC 2845
	case 256:
		return "URI" // RFC 7553
	case 257:
		return "CAA" // RFC 6844
	case 32768:
		return "TA" // — 	DNSSEC Trust Authorities
	case 32769:
		return "DLV" // RFC 4431
	default:
		return "?????"
	}
}

// RFC 1035
type Header struct {
	Id uint16 // identification number

	RD     bool  // recursion desired
	TC     bool  // truncated message
	AA     bool  // authoritive answer
	OpCode uint8 // purpose of message
	QR     bool  // query/response flag

	RCode uint8 // response code
	CD    bool  // checking disabled
	AD    bool  // authenticated data
	Z     bool  // reserved
	RA    bool  // recursion available

	Questions   uint16 // number of question entries
	Answers     uint16 // number of answer entries
	Authorities uint16 // number of authority entries
	Additionals uint16 // number of resource entries
}

func read_header(buf []byte, off int, header *Header) (int, error) {
	if off+12 >= len(buf) {
		return 0, fmt.Errorf("Out of bounds")
	}
	off, _ = binary.Read16(buf, off, &header.Id)

	var flags uint8
	off, _ = binary.Read8(buf, off, &flags)
	header.RD = (flags & 0x01) > 0
	header.TC = (flags & 0x02) > 0
	header.AA = (flags & 0x04) > 0
	header.OpCode = (flags >> 3) & 0x0F
	header.QR = (flags & 0x80) > 0

	off, _ = binary.Read8(buf, off, &flags)
	header.RCode = flags & 0x0F
	header.CD = (flags & 0x10) > 0
	header.AD = (flags & 0x20) > 0
	header.Z = (flags & 0x40) > 0
	header.RA = (flags & 0x80) > 0

	off, _ = binary.Read16(buf, off, &header.Questions)
	off, _ = binary.Read16(buf, off, &header.Answers)
	off, _ = binary.Read16(buf, off, &header.Authorities)
	off, _ = binary.Read16(buf, off, &header.Additionals)

	return off, nil
}

type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

type Record struct {
	Name  string
	Type  uint16
	Class uint16
	Ttl   uint32
	Rdlen uint16
	Rdata uint8
}

type Message struct {
	Header     Header
	Question   []Question
	Answer     []Record
	Authority  []Record
	Additional []Record
}

func ParseMessage(buf []byte) (*Message, error) {
	m := Message{}
	m.Header = Header{}
	m.Question = make([]Question, 0, 1)
	m.Answer = make([]Record, 0, 1)
	m.Authority = make([]Record, 0)
	m.Additional = make([]Record, 0)

	off, err := read_header(buf, 0, &m.Header)
	if err != nil {
		return nil, err
	}
	if m.Header.Questions == 1 {
		question := Question{Name: ""}
		off, err = read_question(buf, off, &question)
		if err != nil {
			return nil, err
		}
		m.Question = append(m.Question, question)
	} else {
		return nil, fmt.Errorf("Query must have one question")
	}
	return &m, nil
}

func ValidateMessage(buf []byte) bool {
	header := Header{}
	_, err := read_header(buf, 0, &header)
	if err != nil {
		fmt.Println("Unable to header DNS header")
		return false
	}
	return header.Questions == 1
}

func read_question(buf []byte, off int, value *Question) (int, error) {
	off, err := read_domain(buf, off, &value.Name)
	if err != nil {
		return 0, err
	}
	off, _ = binary.Read16(buf, off, &value.Type)
	off, _ = binary.Read16(buf, off, &value.Class)
	return off, nil
}

func read_domain(buf []byte, off int, value *string) (int, error) {
	if off >= len(buf) {
		return 0, fmt.Errorf("Out of bounds")
	}
	cur := off
	out := make([]byte, 0, 12)

	for buf[cur] != 0 {
		// check whether we have a pointer (RFC-1035 4.1.4. Message compression)
		if (buf[cur] & 0xC0) == 0xC0 {
			if cur+3 >= len(buf) {
				return 0, fmt.Errorf("Out of bounds")
			}
			cur = (int(buf[cur]&0x3F) << 8) | int(buf[cur+1])
			if cur >= len(buf) {
				return 0, fmt.Errorf("Out of bounds")
			}
			_, err := read_domain(buf, cur, value)
			return cur, err
		}

		// extract current group
		size := int(buf[cur] & 0x3F)
		cur++
		for i := 0; i < size; i++ {
			c := buf[cur]
			if c >= 'A' && c <= 'Z' {
				c += 32
			}
			out = append(out, c)
			cur++
		}
		if buf[cur] != 0 {
			out = append(out, '.')
		}
	}

	*value = *value + string(out)
	return cur + 1, nil
}

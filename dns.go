package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func _read_u8(reader *bytes.Reader) (uint8, error) {
	var value uint8
	result := binary.Read(reader, binary.BigEndian, &value)
	if result == nil {
		return value, nil
	} else {
		return 0, result
	}
}

func _read_u16(reader *bytes.Reader) (uint16, error) {
	var value uint16
	result := binary.Read(reader, binary.BigEndian, &value)
	if result == nil {
		return value, nil
	} else {
		return 0, result
	}
}

func _read_u32(reader *bytes.Reader) (uint32, error) {
	var value uint32
	result := binary.Read(reader, binary.BigEndian, &value)
	if result == nil {
		return value, nil
	} else {
		return 0, result
	}
}

func rcode_to_string(value int) string {
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

func type_to_string(value int) string {
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
type dns_header struct {
	id uint16 // identification number

	rd     bool  // recursion desired
	tc     bool  // truncated message
	aa     bool  // authoritive answer
	opcode uint8 // purpose of message
	qr     bool  // query/response flag

	rcode uint8 // response code
	cd    bool  // checking disabled
	ad    bool  // authenticated data
	z     bool  // reserved
	ra    bool  // recursion available

	qst_count  uint16 // number of question entries
	ans_count  uint16 // number of answer entries
	auth_count uint16 // number of authority entries
	add_count  uint16 // number of resource entries
}

func read_header(reader *bytes.Reader, header *dns_header) bool {
	header.id, _ = _read_u16(reader)

	flags, _ := _read_u8(reader)
	header.rd = (flags & 0x01) > 0
	header.tc = (flags & 0x02) > 0
	header.aa = (flags & 0x04) > 0
	header.opcode = (flags >> 3) & 0x0F
	header.qr = (flags & 0x80) > 0

	flags, _ = _read_u8(reader)
	header.rcode = flags & 0x0F
	header.cd = (flags & 0x10) > 0
	header.ad = (flags & 0x20) > 0
	header.z = (flags & 0x40) > 0
	header.ra = (flags & 0x80) > 0

	header.qst_count, _ = _read_u16(reader)
	header.ans_count, _ = _read_u16(reader)
	header.auth_count, _ = _read_u16(reader)
	header.add_count, _ = _read_u16(reader)
	return true
}

func ValidateMessage(msg []byte) bool {
	reader := bytes.NewReader(msg)
	header := dns_header{}
	result := read_header(reader, &header)
	if !result {
		fmt.Println("unable to header DNS header")
	}
	fmt.Println(msg)
	fmt.Println(header)
	return header.qst_count == 1
}

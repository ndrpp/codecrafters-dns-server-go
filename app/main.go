package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

type Record struct {
	Name  string
	Type  uint16
	Class uint16
	TTL   uint32
	Len   uint16
	Data  string
}

type ResultCode int

const (
	NOERROR ResultCode = iota
	FORMERR
	SERVFAIL
	NXDOMAIN
	NOTIMP
	REFUSED
)

type DNSHeader struct {
	Id uint16

	Recursion_desired    bool  // 1 bit
	Truncated_message    bool  // 1 bit
	Authoritative_answer bool  // 1 bit
	Opcode               uint8 // 4 bits
	Response             bool  // 1 bit

	Rescode             ResultCode // 4 bits
	Checking_disabled   bool       // 1 bit
	Authed_data         bool       // 1 bit
	Z                   bool       // 1 bit
	Recursion_available bool       // 1 bit

	Questions             uint16 // 16 bits
	Answers               uint16 // 16 bits
	Authoritative_entries uint16 // 16 bits
	Resource_entries      uint16 // 16 bits
}

func NewDNSHeader() DNSHeader {
	return DNSHeader{
		Id: 0,

		Response:             false,
		Opcode:               0,
		Authoritative_answer: false,
		Truncated_message:    false,
		Recursion_desired:    false,
		Recursion_available:  false,
		Z:                    false,
		Rescode:              NOERROR,

		Questions:             0,
		Answers:               0,
		Authoritative_entries: 0,
		Resource_entries:      0,

		Checking_disabled: false,
		Authed_data:       false,
	}
}

type DNSMessage struct {
	Header     DNSHeader
	Question   []Question
	Answer     []Record
	Authority  []Record
	Additional []Record
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		response := HandleReceivedData(receivedData)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

func HandleReceivedData(data string) []byte {
	header := parseHeader(data)
	buf := BuildHeader(header)

	question := Question{
		Name:  "\x0ccodecrafters\x02io\x00",
		Type:  1,
		Class: 1,
	}
	buf = append(buf, BuildQuestion(question)...)

	answer := Record{
		Name:  "\x0ccodecrafters\x02io\x00",
		Type:  1,
		Class: 1,
		TTL:   60,
		Len:   4,
		Data:  "\x08\x08\x08\x08",
	}
	buf = append(buf, BuildAnswer(answer)...)

	return buf
}

func parseHeader(data string) DNSHeader {
	bd := []byte(data)

	header := NewDNSHeader()
	header.Id = binary.BigEndian.Uint16(bd[0:2])
	header.Questions = binary.BigEndian.Uint16(bd[4:6])
	header.Answers = binary.BigEndian.Uint16(bd[4:6])

	flags := binary.BigEndian.Uint16(bd[2:4])
	header.Response = true
	header.Authoritative_answer = false
	header.Truncated_message = false
	header.Opcode = uint8((flags & 0x7800) >> 11)
	header.Recursion_desired = (flags & 0x0100) != 0

	header.Recursion_available = false
	header.Z = false
	header.Rescode = NOTIMP

	return header
}

func BuildAnswer(r Record) []byte {
	var b bytes.Buffer
	w := io.Writer(&b)

	w.Write([]byte(r.Name))
	binary.Write(w, binary.BigEndian, r.Type)
	binary.Write(w, binary.BigEndian, r.Class)
	binary.Write(w, binary.BigEndian, r.TTL)
	binary.Write(w, binary.BigEndian, r.Len)
	w.Write([]byte(r.Data))

	return b.Bytes()
}

func BuildQuestion(question Question) []byte {
	var b bytes.Buffer
	w := io.Writer(&b)

	w.Write([]byte(question.Name))
	binary.Write(w, binary.BigEndian, question.Type)
	binary.Write(w, binary.BigEndian, question.Class)

	return b.Bytes()
}

func BuildHeader(header DNSHeader) []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[0:2], header.Id)
	binary.BigEndian.PutUint16(buf[2:4], BuildFlags(header))
	binary.BigEndian.PutUint16(buf[4:6], header.Questions)
	binary.BigEndian.PutUint16(buf[6:8], header.Answers)

	return buf
}

func BuildFlags(header DNSHeader) uint16 {
	var flags uint16
	if header.Response {
		flags |= 0x8000
	}
	flags |= uint16(header.Opcode) << 11
	if header.Authoritative_answer {
		flags |= 0x0400
	}
	if header.Truncated_message {
		flags |= 0x0200
	}
	if header.Recursion_desired {
		flags |= 0x0100
	}
	if header.Recursion_available {
		flags |= 0x0080
	}
	if header.Z == true {
		flags |= uint16(1) << 4
	} else {
		flags |= uint16(0) << 4
	}
	flags |= uint16(header.Rescode)
	return flags
}

package main

type Opcode byte

const (
	Continuation Opcode = 0x0
	Text         Opcode = 0x1
	Binary       Opcode = 0x2
	Close        Opcode = 0x8
	Ping         Opcode = 0x9
	Pong         Opcode = 0xA
)

type Frame struct {
	Opcode   Opcode
	Payload  []byte
	IsFinal  bool
}

func (f Frame) SetOpcode(b byte) {
	f.Opcode = Opcode(b)
}

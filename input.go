package main

import (
	"fmt"
	"github.com/tarm/serial"
	"math/rand"
	"time"
)

type Input interface {
	ReadLine() (error, []byte)
	Close()
}

// UART input

//noinspection SpellCheckingInspection
type UART struct {
	port *serial.Port
}

func OpenUART(device string, baudRate int) (error, *UART) {
	c := &serial.Config{Name: device, Baud: baudRate}

	port, err := serial.OpenPort(c)
	if err != nil {
		return fmt.Errorf("Can't open serial device %s: %v", device, err), nil
	}

	return nil, &UART{port: port}
}

func (u *UART) ReadLine() (error, []byte) {
	buf := make([]byte, 0)
	idx := 0
	for {
		charBuf := make([]byte, 1)
		n, err := u.port.Read(charBuf)
		if err != nil || n != 1 {
			return err, nil
		}

		ch := charBuf[0]
		buf = append(buf, ch)
		idx++

		if len(buf) > 2 && buf[len(buf)-2] == '\r' && buf[len(buf)-1] == '\n' {
			return nil, buf[:idx]
		}
	}
}

func (u *UART) Close() {
	u.port.Close()
}

// Random input

type RandomInput struct {
	src *rand.Rand
	ts  time.Time
}

func NewRandomInput() *RandomInput {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	ts := time.Now().Add(-11 * time.Second)
	return &RandomInput{src: r, ts: ts}
}

func getRandomData(in *RandomInput) []byte {
	temp := in.src.Intn(1000)
	rhTrue := in.src.Intn(9000)
	press := in.src.Intn(13000)
	cups := in.src.Intn(150)

	data := fmt.Sprintf("{ \"weather\": { \"station-id\": 1, \"temp_m\": %d, "+
		"\"rh-true_m\": %d, \"pressure-nn_c\": %d, \"rain-cupfills\": %d } }",
		temp, rhTrue, press, cups)

	return []byte(data)
}

func (in *RandomInput) ReadLine() (error, []byte) {
	for !time.Now().After(in.ts.Add(10 * time.Second)) {
		time.Sleep(2 * time.Second)
	}
	in.ts = time.Now()
	data := getRandomData(in)
	return nil, data
}

func (in *RandomInput) Close() {}

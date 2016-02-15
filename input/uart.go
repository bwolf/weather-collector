package input

import (
	"fmt"
	"github.com/tarm/serial"
)

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

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/exp/io/i2c"
)

const (
	devFile       = "/dev/i2c-4"
	address       = 0x68
	accelRegister = 0x3B
)

// ref. https://godoc.org/golang.org/x/exp/io/i2c
// ref. https://github.com/tinygo-org/drivers/tree/master/mpu6050

func main() {
	// create a unix domain socket
	path := filepath.Join(os.TempDir(), "gsensord-socket")
	os.Remove(path)
	listener, _ := net.Listen("unix", path)
	defer listener.Close()

	// start mpu6050
	d, err := i2c.Open(&i2c.Devfs{Dev: devFile}, address)
	if err != nil {
		panic(err)
	}
	defer d.Close()
	d.WriteReg(0x6B, []uint8{0})

	for {
		log.Println("Listening...")
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		log.Println("connected")

		go func() {
			defer log.Println("closed")
			defer conn.Close()

			// main loop
			for {
				data := make([]byte, 6)
				if err := d.ReadReg(accelRegister, data); err != nil {
					panic(err)
				}

				// ref. https://github.com/tinygo-org/drivers/tree/master/mpu6050
				x := int32(int16((uint16(data[0])<<8)|uint16(data[1]))) * 15625 / 256
				y := int32(int16((uint16(data[2])<<8)|uint16(data[3]))) * 15625 / 256
				z := int32(int16((uint16(data[4])<<8)|uint16(data[5]))) * 15625 / 256

				gx := float32(x) / 1000
				gy := float32(y) / 1000
				gz := float32(z) / 1000

				s := fmt.Sprintf("a,%.2f,%.2f,%.2f\n", gx, gy, gz)

				_, err = conn.Write([]byte(s))
				if err != nil {
					log.Printf("error: %v\n", err)
					return
				}

				time.Sleep(100 * time.Millisecond)
			}

		}()
	}

}

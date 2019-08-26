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
	devFile       = "/dev/i2c-2"
	address       = 0x68
	accelRegister = 0x3B
)

// ref. https://godoc.org/golang.org/x/exp/io/i2c
// ref. https://github.com/tinygo-org/drivers/tree/master/mpu6050

func abs(val int32) int32 {
	if val < 0 {
		return -val
	}
	return val
}

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
			lastSend := time.Now()
			var maxGx, maxGy, maxGz int32
			for {
				data := make([]byte, 6)
				if err := d.ReadReg(accelRegister, data); err != nil {
					panic(err)
				}

				// ref. https://github.com/tinygo-org/drivers/tree/master/mpu6050
				microGx := int32(int16((uint16(data[0])<<8)|uint16(data[1]))) * 15625 / 256
				microGy := int32(int16((uint16(data[2])<<8)|uint16(data[3]))) * 15625 / 256
				microGz := int32(int16((uint16(data[4])<<8)|uint16(data[5]))) * 15625 / 256

				if abs(microGx) > abs(maxGx) {
					maxGx = microGx
				}
				if abs(microGy) > abs(maxGy) {
					maxGy = microGy
				}
				if abs(microGz) > abs(maxGz) {
					maxGz = microGz
				}

				if time.Now().Sub(lastSend) > 100*time.Millisecond {
					gx := float32(maxGx) / 1000
					gy := float32(maxGy) / 1000
					gz := float32(maxGz) / 1000

					s := fmt.Sprintf("a,%.2f,%.2f,%.2f\n", gx, gy, gz)

					_, err = conn.Write([]byte(s))
					if err != nil {
						log.Printf("error: %v\n", err)
						return
					}

					maxGx = 0
					maxGy = 0
					maxGz = 0
					lastSend = time.Now()
				}
			}

		}()
	}

}

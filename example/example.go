package main

import (
	"fmt"
	"log"
	"time"

	"github.com/MasandeM/sps30"

	"go.bug.st/serial"
)

func main() {

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	log.Println("Connecting to UART")
	var err error
	uart, err := serial.Open("/dev/ttyUSB0", mode) //should be read from a config file or something
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully Connected")

	device := sps30.New(uart)

	//Create a struct that is passed to a function then is populated. th
	version_info := sps30.VersionInfo{}
	ret := device.ReadVersion(&version_info)

	if ret != 0 {
		log.Fatal("Error reading version information: ", ret)
	}

	fmt.Printf("FW: %d.%d, HW: %d, SHDLC: %d.%d\n",
		version_info.FirmwarMajor,
		version_info.FirmwarMinor,
		version_info.HardwarRevision,
		version_info.SHDLCMajor,
		version_info.SHDLCMinor)

	measurement := sps30.Measurement{}
	for {
		ret = device.StartMeasurement()
		if ret != 0 {
			fmt.Println("error starting measurement")
		}

		fmt.Println("measurment started")
		ret = device.ReadMeasurement(&measurement)
		if ret != 0 {
			fmt.Println("error reading measurement")
		} else {
			fmt.Printf(`measured values:
				%0.2f pm1.0
				%0.2f pm2.5
				%0.2f pm4.0
				%0.2f pm10.0
				%0.2f nc0.5
				%0.2f nc1.0
				%0.2f nc2.5
				%0.2f nc4.5
				%0.2f nc10.0
				%0.2f typical particle size
	`,
				measurement.Mc1p0, measurement.Mc2p5, measurement.Mc4p0, measurement.Mc10p0, measurement.Nc0p5,
				measurement.Nc1p0, measurement.Nc2p5, measurement.Nc4p0, measurement.Nc10p0,
				measurement.TypicalParticleSize)
		}

		time.Sleep(time.Second)
	}
}

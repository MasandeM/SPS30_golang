package sps30

import (
	"encoding/binary"
	"log"
	"math"

	"go.bug.st/serial"
)

const SHDLC_START = 0x7e
const SHDLC_STOP = 0x7e
const SHDLC_FRAME_MAX_TX_FRAME_SIZE = 520 // start/stop + (4 header + 255 data) * 2 for byte stuffing
const SHDLC_FRAME_MAX_RX_FRAME_SIZE = 522 // start/stop + (5 header + 255 data) * 2 because of byte stuffing

const SHDLC_ERR_NO_DATA = -1
const SHDLC_ERR_MISSING_START = -2
const SHDLC_ERR_MISSING_STOP = -3
const SHDLC_ERR_CRC_MISMATCH = -4
const SHDLC_ERR_ENCODING_ERROR = -5
const SHDLC_ERR_TX_INCOMPLETE = -6
const SHDLC_ERR_FRAME_TOO_LONG = -7
const SHDLC_RX_HEADER_SIZE = 4
const PERIPHERAL_ADDR = 0
const CMD_READ_VERSION = 0xd1
const CMD_START_MEASUREMENT = 0x00
const CMD_READ_MEASUREMENT = 0x03
const CMD_WAKE_UP = 0x11
const ERR_NOT_ENOUGH_DATA = -1

var SUBCMD_MEASUREMENT_START = [2]byte{0x01, 0x03}

type SHDLCRxHeader struct {
	addr     uint8
	cmd      uint8
	state    uint8
	data_len uint8
}

type VersionInfo struct {
	FirmwarMajor    uint8
	FirmwarMinor    uint8
	HardwarRevision uint8
	SHDLCMajor      uint8
	SHDLCMinor      uint8
}

type Measurement struct {
	Mc1p0               float32
	Mc2p5               float32
	Mc4p0               float32
	Mc10p0              float32
	Nc0p5               float32
	Nc1p0               float32
	Nc2p5               float32
	Nc4p0               float32
	Nc10p0              float32
	TypicalParticleSize float32
}

type Device struct {
	uart serial.Port
}

var port serial.Port

func New(uart serial.Port) Device {
	return Device{
		uart: uart,
	}
}

func (d *Device) Wakeup() {
	data := []byte{0xff}
	log.Println("sending data")
	_, err := d.uart.Write(data)
	log.Println("complete sending")
	if err != nil {
		log.Fatal(err)
	}
	rx_header := SHDLCRxHeader{}
	rdata := make([]byte, 0)

	d.SHDLCTransmitReceive(PERIPHERAL_ADDR, CMD_WAKE_UP, 0, nil, 0, &rx_header, &rdata)
}

func (d *Device) ReadVersion(version_info *VersionInfo) int {

	rx_header := SHDLCRxHeader{}
	data := make([]byte, 7)

	ret := d.SHDLCTransmitReceive(PERIPHERAL_ADDR, CMD_READ_VERSION, 0, nil, uint8(len(data)), &rx_header, &data)

	if ret != 0 {
		return ret
	}

	if int(rx_header.data_len) != len(data) {
		return ERR_NOT_ENOUGH_DATA
	}

	if rx_header.state != 0 {
		return int(rx_header.state) //original code uses masking. Is this needed?
	}

	version_info.FirmwarMajor = data[0]
	version_info.FirmwarMinor = data[1]
	version_info.HardwarRevision = data[3]
	version_info.SHDLCMajor = data[5]
	version_info.SHDLCMinor = data[6]

	return 0
}

func (d *Device) StartMeasurement() int {
	rx_header := SHDLCRxHeader{}
	subcmd := []byte{0x01, 0x03}
	data := make([]byte, 0)

	return d.SHDLCTransmitReceive(PERIPHERAL_ADDR, CMD_START_MEASUREMENT, uint8(len(subcmd)), subcmd, 0, &rx_header, &data)
}

func (d *Device) ReadMeasurement(measurement *Measurement) int {
	rx_header := SHDLCRxHeader{}
	data := make([]byte, 40)

	ret := d.SHDLCTransmitReceive(PERIPHERAL_ADDR, CMD_READ_MEASUREMENT, 0, nil, uint8(len(data)), &rx_header, &data)

	if ret != 0 {
		return ret
	}

	if int(rx_header.data_len) != len(data) {
		return ERR_NOT_ENOUGH_DATA
	}

	(*measurement).Mc1p0 = BytesFloat32(data[0:4])
	(*measurement).Mc2p5 = BytesFloat32(data[4:8])
	(*measurement).Mc4p0 = BytesFloat32(data[8:12])
	(*measurement).Mc10p0 = BytesFloat32(data[12:16])
	(*measurement).Nc0p5 = BytesFloat32(data[16:20])
	(*measurement).Nc1p0 = BytesFloat32(data[20:24])
	(*measurement).Nc2p5 = BytesFloat32(data[24:28])
	(*measurement).Nc4p0 = BytesFloat32(data[28:32])
	(*measurement).Nc10p0 = BytesFloat32(data[32:36])
	(*measurement).TypicalParticleSize = BytesFloat32(data[36:40])

	if rx_header.state != 0 {
		return int(rx_header.state)
	}

	return 0
}

func BytesFloat32(bytes []byte) float32 {
	bits := binary.BigEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func (d *Device) SHDLCTransmitReceive(addr uint8,
	cmd uint8, tx_data_len uint8,
	tx_data []byte,
	max_rx_data_len uint8,
	rx_header *SHDLCRxHeader,
	rx_data *[]byte) int {
	// transcieve (transmit then receive) and SHDLC Frame

	ret := d.shdlc_tx(addr, cmd, tx_data_len, tx_data)
	if ret != 0 {
		log.Fatal("Failed to send data to sensor")
		return ret
	}

	return d.shdlc_rx(int(max_rx_data_len), rx_header, rx_data)
}

func (d *Device) shdlc_tx(addr uint8, cmd uint8, data_len uint8, data []byte) int {

	var tx_frame = [SHDLC_FRAME_MAX_TX_FRAME_SIZE]byte{}
	len := 0

	crc := shdlc_crc(addr+cmd, data_len, data)

	tx_frame[len] = SHDLC_START
	len += 1

	len = shdlc_stuff_data(1, []byte{addr}, &tx_frame, len)
	len = shdlc_stuff_data(1, []byte{cmd}, &tx_frame, len)
	len = shdlc_stuff_data(1, []byte{data_len}, &tx_frame, len)
	len = shdlc_stuff_data(int(data_len), data, &tx_frame, len)

	tx_frame[len] = crc
	len += 1

	tx_frame[len] = SHDLC_STOP
	len += 1

	_, err := d.uart.Write(tx_frame[:len])
	if err != nil {
		log.Fatal(err)
		return -1
	}

	return 0
}

func (d *Device) shdlc_rx(max_data_len int, rx_header *SHDLCRxHeader, data *[]byte) int {

	rx_frame := make([]byte, SHDLC_FRAME_MAX_RX_FRAME_SIZE)
	header_index := 0
	data_index := 0
	var crc uint8

	frame_len, err := d.uart.Read(rx_frame)
	if err != nil {
		log.Fatal(err)
	}

	if frame_len < 1 || rx_frame[0] != SHDLC_START {
		return SHDLC_ERR_MISSING_START
	}

	// get Frame Header
	header_index = 1
	header_index = unstuff_byte(rx_frame, header_index, &(*rx_header).addr)
	header_index = unstuff_byte(rx_frame, header_index, &(*rx_header).cmd)
	header_index = unstuff_byte(rx_frame, header_index, &(*rx_header).state)
	header_index = unstuff_byte(rx_frame, header_index, &(*rx_header).data_len)

	data_index = header_index
	i := 0
	for data_index < frame_len-2 && i < max_data_len {
		data_index = unstuff_byte(rx_frame, data_index, &(*data)[i])
		i += 1
	}

	crc = shdlc_crc(rx_header.addr+rx_header.cmd+rx_header.state, rx_header.data_len, *data)
	if crc != rx_frame[data_index] {
		return SHDLC_ERR_CRC_MISMATCH
	}
	data_index += 1

	if data_index >= frame_len || rx_frame[data_index] != SHDLC_STOP {
		log.Fatal("Missing SHDLC STOP byte")
		return SHDLC_ERR_CRC_MISMATCH
	}

	return 0
}

func unstuff_byte(data []byte, index int, value *uint8) int {

	if data[index] == 0x7d {
		switch data[index+1] {
		case 0x31:
			(*value) = 0x11
		case 0x33:
			(*value) = 0x13
		case 0x5D:
			(*value) = 0x7D
		case 0x5E:
			(*value) = 0x7E
		default:
			(*value) = data[index+1]
		}
		index += 2
		return index

	} else {
		(*value) = data[index]
		index += 1
		return index
	}
}

func shdlc_stuff_data(data_len int, data []byte, stuffed_data *[SHDLC_FRAME_MAX_TX_FRAME_SIZE]byte, index int) int {

	for i := 0; i < data_len; i++ {
		switch data[i] {
		case 0x11:
		case 0x13:
		case 0x7D:
		case 0x7E:
			(*stuffed_data)[index] = 0x7D
			(*stuffed_data)[index+1] = 0x5E
			index += 2
		default:
			(*stuffed_data)[index] = data[i]
			index += 1
		}

	}
	return index
}

func shdlc_crc(header_sum uint8, data_len uint8, data []byte) uint8 {

	sum := header_sum + data_len

	for i := 0; i < int(data_len); i++ {
		sum += data[i]
	}

	return ^sum
}

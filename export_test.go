package sps30

var ShdlcCRC = shdlcCRC
var StuffData = stuffData
var UnstuffByte = unstuffByte

type ShdlcRxHeader = shdlcRxHeader

const ShdlcFrameMaxTxFrameSize = shdlcFrameMaxTxFrameSize

func (d *Device) ShdlcRx(max_data_len int, rx_header *shdlcRxHeader, data *[]byte) error {
	return d.shdlcRx(max_data_len, rx_header, data)
}

func (d *Device) ShdlcTx(addr uint8, cmd uint8, data_len uint8, data []byte) error {
	return d.shdlcTx(addr, cmd, data_len, data)
}

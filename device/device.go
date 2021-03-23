package device

import (
	"fmt"
	"github.com/omzlo/i2cflash/i2c"
	"time"
)

const (
	ADDR_MARKER     = 0
	ADDR_MCUID      = 4
	ADDR_PAGE_SIZE  = 8
	ADDR_FLASH_SIZE = 10
	ADDR_PROG_START = 12
	ADDR_VERSION    = 16
	ADDR_ERR        = 18
	ADDR_PROG       = 19
	ADDR_ADDR       = 20
	ADDR_DATA       = 24
	MAX_DATA_LEN    = 128
)

const (
	PROG_NONE       = 0
	PROG_ERASE_PAGE = 1
	PROG_READ       = 2
	PROG_WRITE      = 3
	PROG_EXIT       = 4
)

type Device struct {
	Address   uint8
	I2cBus    i2c.Bus
	McuId     uint32
	PageSize  uint16
	FlashSize uint16
	ProgStart uint32
	Version   uint8
}

func Open(i2c_bus int, i2c_addr uint8) (*Device, error) {
	var req [17]byte

	bus := i2c.OpenBus(i2c_bus)
	if bus < 0 {
		return nil, fmt.Errorf("Failed to open I2C bus %d", i2c_bus)
	}

	if err := bus.ReadBytes(i2c_addr, ADDR_MARKER, req[:4]); err != nil {
		bus.CloseBus()
		return nil, err
	}
	if req[0] != 'B' || req[1] != 'O' || req[2] != 'O' || req[3] != 'T' {
		bus.CloseBus()
		return nil, fmt.Errorf("Incorrect device signature at address 0: %q", req[:4])
	}

	if err := bus.ReadBytes(i2c_addr, ADDR_MARKER, req[:]); err != nil {
		bus.CloseBus()
		return nil, err
	}

	McuId := uint32(req[4]) | (uint32(req[5]) << 8) | (uint32(req[6]) << 16) | (uint32(req[7]) << 24)
	PageSize := uint16(req[8]) | (uint16(req[9]) << 8)
	FlashSize := uint16(req[10]) | (uint16(req[11]) << 8)
	ProgStart := uint32(req[12]) | (uint32(req[13]) << 8) | (uint32(req[14]) << 16) | (uint32(req[15]) << 24)
	Version := uint8(req[16])

	return &Device{Address: i2c_addr, I2cBus: bus, McuId: McuId, PageSize: PageSize, FlashSize: FlashSize, ProgStart: ProgStart, Version: Version}, nil
}

func (dev *Device) Close() {
	dev.I2cBus.CloseBus()
}

func (dev *Device) WriteBytes(reg byte, data []byte) error {
	for {
		rem := len(data)
		if rem <= 32 {
			return dev.I2cBus.WriteBytes(dev.Address, reg, data)
		}
		if err := dev.I2cBus.WriteBytes(dev.Address, reg, data[:32]); err != nil {
			return err
		}
		data = data[32:]
		reg += 32
	}
}

func (dev *Device) WriteByte(reg byte, data byte) error {
	return dev.I2cBus.WriteByte(dev.Address, reg, data)
}

func (dev *Device) ReadBytes(reg byte, data []byte) error {
	for {
		rem := len(data)
		if rem <= 32 {
			return dev.I2cBus.ReadBytes(dev.Address, reg, data)
		}
		if err := dev.I2cBus.ReadBytes(dev.Address, reg, data[:32]); err != nil {
			return err
		}
		data = data[32:]
		reg += 32
	}
}

func (dev *Device) ReadByte(reg byte) (byte, error) {
	return dev.I2cBus.ReadByte(dev.Address, reg)
}

func (dev *Device) ErrorStatusByte() byte {
	res, err := dev.ReadByte(ADDR_ERR)
	if err != nil {
		return 0x80
	}
	return res
}

func (dev *Device) FlashSetAddress(addr uint32) error {
	var req [4]byte
	req[0] = byte(addr)
	req[1] = byte(addr >> 8)
	req[2] = byte(addr >> 16)
	req[3] = byte(addr >> 24)
	return dev.WriteBytes(ADDR_ADDR, req[:])
}

func (dev *Device) FlashGetAddress() (uint32, error) {
	var buf [4]byte
	err := dev.ReadBytes(ADDR_ADDR, buf[:])
	if err != nil {
		return 0, err
	}
	return (uint32(buf[0])) + (uint32(buf[1]) << 8) + (uint32(buf[2]) << 16) + (uint32(buf[3]) << 24), nil
}

func (dev *Device) waitProg(prog byte, gap time.Duration) error {
	if err := dev.WriteByte(ADDR_PROG, prog); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		time.Sleep(gap)
		response, err := dev.ReadByte(ADDR_PROG)
		if err != nil {
			return err
		}
		if response == 0 {
			return nil
		}
	}
	return fmt.Errorf("Timed out waiting for PROG code 0x%02x to execute.", prog)
}

func (dev *Device) AutoTest() error {
	var buf [64]byte

	for i := 0; i < 64; i++ {
		buf[i] = byte(63 - i)
	}
	if err := dev.WriteBytes(ADDR_DATA, buf[:]); err != nil {
		return err
	}
	if err := dev.ReadBytes(ADDR_DATA, buf[:]); err != nil {
		return err
	}
	for i := 0; i < 64; i++ {
		if buf[i] != byte(63-i) {
			return fmt.Errorf("Inconsistent autotest at byte %d", i)
		}
	}
	return nil
}

func (dev *Device) FlashRead(data []byte) error {
	var buf [4096]byte

	psize := int(dev.PageSize)

	fmt.Printf("Reading %d bytes from address 0x%08x in pages of %d bytes each\n", len(data), dev.ProgStart, psize)
	if err := dev.FlashSetAddress(dev.ProgStart); err != nil {
		return err
	}

	fmt.Printf("Reading: [")
	defer fmt.Printf("\n")
	for pos := 0; pos < len(data); pos += psize {
		if err := dev.waitProg(PROG_READ, 1*time.Millisecond); err != nil {
			return err
		}
		if err := dev.ReadBytes(ADDR_DATA, buf[:psize]); err != nil {
			return err
		}
		copy(data[pos:], buf[:psize])
		fmt.Print("#")
	}
	fmt.Print("]")
	return nil
}

func (dev *Device) flashErase(block_count int) error {
	fmt.Printf("Erasing: [")
	defer fmt.Print("\n")

	for pos := 0; pos < block_count; pos++ {
		addr := dev.ProgStart + uint32(pos)*uint32(dev.PageSize)
		fmt.Print("#")
		if err := dev.FlashSetAddress(addr); err != nil {
			return err
		}
		if err := dev.waitProg(PROG_ERASE_PAGE, 50*time.Millisecond); err != nil {
			return err
		}
		if ecode := dev.ErrorStatusByte(); ecode != 0 {
			return fmt.Errorf("Flash erase failed at 0x%x for block %d/%d with error code 0x%02x", addr, pos, block_count, ecode)
		}
	}
	fmt.Print("]")
	return nil
}

func (dev *Device) flashWrite(data []byte) error {
	var buf [4096]byte

	psize := int(dev.PageSize)

	fmt.Printf("Writing: [")
	defer fmt.Print("\n")

	for pos := 0; pos < len(data); pos += psize {
		copy(buf[:psize], data[pos:])

		if err := dev.WriteBytes(ADDR_DATA, buf[:psize]); err != nil {
			return err
		}
		if err := dev.waitProg(PROG_WRITE, 3*time.Millisecond); err != nil {
			return err
		}
		if ecode := dev.ErrorStatusByte(); ecode != 0 {
			return fmt.Errorf("Flash write failed, error code 0x%02x", ecode)
		}
		fmt.Print("#")
	}
	fmt.Print("]")

	return nil
}

func (dev *Device) FlashWrite(data []byte) error {

	if err := dev.flashErase((len(data) + (int(dev.PageSize) - 1)) / int(dev.PageSize)); err != nil {
		return err
	}

	if err := dev.FlashSetAddress(dev.ProgStart); err != nil {
		return err
	}

	if err := dev.flashWrite(data); err != nil {
		return err
	}

	data_copy := make([]byte, len(data))
	if err := dev.FlashRead(data_copy); err != nil {
		return err
	}
	for i := 0; i < len(data); i++ {
		if data[i] != data_copy[i] {
			start := i & (^0xf)
			fmt.Printf("File 0x%08x:", uint32(start)+dev.ProgStart)
			for j := start; (j < start+16) && (j < len(data)); j++ {
				fmt.Printf(" %02x", data[j])
			}
			fmt.Println()
			fmt.Printf("Read 0x%08x:", uint32(start)+dev.ProgStart)
			for j := start; (j < start+16) && (j < len(data)); j++ {
				fmt.Printf(" %02x", data_copy[i])
			}
			fmt.Println()
			fmt.Printf("Error status byte is %02x\n", dev.ErrorStatusByte())
			return fmt.Errorf("Inconsistent flash at byte %x (start+%d), expected 0x%02x, found 0x%02x", dev.ProgStart+uint32(i), i, data[i], data_copy[i])
		}
	}
	fmt.Println("Flash content verified.")
	return nil
}

func (dev *Device) FlashExit() error {
	return dev.WriteByte(ADDR_PROG, PROG_EXIT)
}

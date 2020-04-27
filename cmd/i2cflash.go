package main

import (
	"flag"
	"fmt"
	"github.com/omzlo/i2cflash/device"
	"io/ioutil"
	"os"
	"strconv"
)

var opt_address uint

const I2CFLASH_VERSION = "1.0"

var HELP_TEXT = `i2cflash [-address=<i2c_addr>] <command> [<args>]
	Where <command> [<args>] can take the following format:
	- info 
		Print info about the device firware.

	- read filename
		Read <size> bytes of firmware in binary format is store it in 'filename'. 
		If <size> is omitted or set to '-1', read all available flash.

	- write filename
		Write the firmware contained in 'filename' into the device's flash.

	- exit
		Leave bootloader mode.

	All commands can optionnly specify an I2C address with the -address option. 
	If no I2C address is specified the default 7-bit address 0x6C is used.
`

func process_command(fs *flag.FlagSet) error {
	var i2c_addr = uint8(opt_address)

	if fs.NArg() == 0 {
		return fmt.Errorf("Type 'i2cflash help' for usage information")
	}

	switch fs.Arg(0) {
	case "read":
		var length int

		if fs.NArg() < 2 {
			return fmt.Errorf("Missing file name parameter")
		}
		if fs.NArg() > 3 {
			return fmt.Errorf("Too many parameters")
		}
		fname := fs.Arg(1)
		if fs.NArg() == 3 {
			l, err := strconv.ParseInt(fs.Arg(2), 0, 32)
			if err != nil {
				return err
			}
			length = int(l)
		} else {
			length = -1
		}

		d, err := device.Open(1, i2c_addr)
		if err != nil {
			return err
		}
		defer d.Close()

		bsize := uint16(((d.ProgStart) & 0xFFFF) / 1024)
		if length < 0 {
			length = int(d.FlashSize-bsize) * 1024
		}
		if length > int(d.FlashSize-bsize)*1024 {
			return fmt.Errorf("Cannot read more than %d bytes from flash", int(d.FlashSize-bsize)*1024)
		}
		buf := make([]byte, length, length)
		if err := d.FlashRead(buf); err != nil {
			return err
		}
		if err := ioutil.WriteFile(fname, buf, 0644); err != nil {
			return err
		}
	case "write":
		if fs.NArg() != 2 {
			return fmt.Errorf("Missing file name parameter")
		}

		fname := fs.Arg(1)

		buf, err := ioutil.ReadFile(fname)
		if err != nil {
			return err
		}

		d, err := device.Open(1, i2c_addr)
		if err != nil {
			return err
		}
		defer d.Close()
		if err := d.FlashWrite(buf); err != nil {
			return err
		}
	case "exit":
		d, err := device.Open(1, i2c_addr)
		if err != nil {
			return err
		}
		defer d.Close()

		if err := d.FlashExit(); err != nil {
			return err
		}
	case "info":
		d, err := device.Open(1, i2c_addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		d.Close()
		fmt.Printf("Bootloader version: %d\n", d.Version)
		fmt.Printf("MCU ID: 0x%08x\n", d.McuId)
		fmt.Printf("Page size: %d\n", d.PageSize)
		fmt.Printf("Flash size (Kb): %d\n", d.FlashSize)
		fmt.Printf("Program start address: 0x%08x\n", d.ProgStart)
		bsize := uint16(((d.ProgStart) & 0xFFFF) / 1024)
		fmt.Printf("Note: Bootloader is %d Kb, leaving %d Kb for program\n", bsize, d.FlashSize-bsize)
	case "autotest":
		d, err := device.Open(1, i2c_addr)
		if err != nil {
			return err
		}
		defer d.Close()

		if err := d.AutoTest(); err != nil {
			return err
		}
	case "help":
		fmt.Printf("i2cflash version %s\nUsage: %s\n", I2CFLASH_VERSION, HELP_TEXT)
	default:
		return fmt.Errorf("Unrecognized subcommand '%s': valid subcommands for flash are 'info', 'read', 'write' and 'exit'.", fs.Arg(0))
	}
	fmt.Println("OK")
	return nil
}

func main() {

	fs := flag.NewFlagSet("i2cflash", flag.ExitOnError)
	fs.UintVar(&opt_address, "address", 0x6c, "Select I2C device address")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	if err := process_command(fs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

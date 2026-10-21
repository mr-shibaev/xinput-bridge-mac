// raikirid reads XInput reports from the ASUS ROG Raikiri II 2.4 GHz dongle
// and publishes them to dll/xinput.c through a shared memory-mapped file.
package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/gousb"
)

const (
	vendorID  = 0x0b05
	productID = 0x1c92

	// statePath is mapped by both this daemon and the DLL (as Z:\tmp\raikiri.xinput).
	statePath = "/tmp/raikiri.xinput"

	// State file layout, little endian, must match shared_t in dll/xinput.c:
	//   0  u32 packet, incremented on every report
	//   4  u8  connected
	//   8  XINPUT_GAMEPAD, 12 bytes copied verbatim from report bytes 2..13
	//   20 u16 rumble left, u16 rumble right, written by the DLL
	stateSize    = 24
	offPacket    = 0
	offConnected = 4
	offGamepad   = 8
	gamepadSize  = 12

	retryDelay = 2 * time.Second
	// wakeInterval throttles the "user is active" declaration; reports arrive
	// on every stick twitch, the idle timer only needs a nudge now and then.
	wakeInterval = 30 * time.Second
)

// applyReport copies a 20-byte Xbox 360 input report into the shared state.
// It returns false if buf is not an input report.
func applyReport(state, buf []byte) bool {
	if len(buf) < 2+gamepadSize || buf[0] != 0x00 || buf[1] != 0x14 {
		return false
	}
	// ponytail: no seqlock, a torn frame of gamepad state is harmless.
	copy(state[offGamepad:offGamepad+gamepadSize], buf[2:2+gamepadSize])
	binary.LittleEndian.PutUint32(state[offPacket:], binary.LittleEndian.Uint32(state[offPacket:])+1)
	return true
}

func main() {
	verbose := flag.Bool("v", false, "print controller state on every report")
	flag.Parse()

	// Open in place, never recreate: the DLL keeps the inode mapped across daemon restarts.
	f, err := os.OpenFile(statePath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(stateSize); err != nil {
		log.Fatal(err)
	}
	state, err := syscall.Mmap(int(f.Fd()), 0, stateSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		log.Fatalf("mmap %s: %v", statePath, err)
	}
	defer syscall.Munmap(state)
	defer func() { state[offConnected] = 0 }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	usb := gousb.NewContext()
	defer usb.Close()

	// Log only on state changes so an unplugged dongle does not flood the log.
	last := ""
	for ctx.Err() == nil {
		err := serve(ctx, usb, state, *verbose)
		state[offConnected] = 0
		if ctx.Err() != nil {
			return
		}
		if msg := err.Error(); msg != last {
			log.Print(msg)
			last = msg
		}
		select {
		case <-ctx.Done():
		case <-time.After(retryDelay):
		}
	}
}

// serve claims the dongle and streams reports into state until the dongle
// goes away or ctx is cancelled. It returns nil only on cancellation.
func serve(ctx context.Context, usb *gousb.Context, state []byte, verbose bool) error {
	dev, err := usb.OpenDeviceWithVIDPID(vendorID, productID)
	if err != nil {
		return fmt.Errorf("open dongle: %w", err)
	}
	if dev == nil {
		return fmt.Errorf("dongle %04x:%04x not found, waiting", vendorID, productID)
	}
	defer dev.Close()

	cfg, err := dev.Config(1)
	if err != nil {
		return err
	}
	defer cfg.Close()
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		return fmt.Errorf("claim XInput interface: %w", err)
	}
	defer intf.Close()

	var in *gousb.InEndpoint
	for _, ep := range intf.Setting.Endpoints {
		if ep.Direction == gousb.EndpointDirectionIn && ep.TransferType == gousb.TransferTypeInterrupt {
			if in, err = intf.InEndpoint(ep.Number); err != nil {
				return err
			}
			break
		}
	}
	if in == nil {
		return fmt.Errorf("no interrupt IN endpoint on interface 0")
	}

	state[offConnected] = 1
	log.Printf("dongle connected, reading %s", in)
	buf := make([]byte, in.Desc.MaxPacketSize)
	var lastWake time.Time
	for {
		n, err := in.ReadContext(ctx, buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("dongle lost: %w", err)
		}
		if !applyReport(state, buf[:n]) {
			if verbose {
				log.Printf("unknown report: % x", buf[:n])
			}
			continue
		}
		if time.Since(lastWake) > wakeInterval {
			lastWake = time.Now()
			if err := declareUserActivity(); err != nil {
				log.Print(err)
			}
		}
		if verbose {
			g := state[offGamepad:]
			fmt.Printf("buttons=%04x lt=%3d rt=%3d lx=%6d ly=%6d rx=%6d ry=%6d\n",
				binary.LittleEndian.Uint16(g), g[2], g[3],
				int16(binary.LittleEndian.Uint16(g[4:])), int16(binary.LittleEndian.Uint16(g[6:])),
				int16(binary.LittleEndian.Uint16(g[8:])), int16(binary.LittleEndian.Uint16(g[10:])))
		}
	}
}

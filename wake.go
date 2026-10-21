package main

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <IOKit/pwr_mgt/IOPMLib.h>

static IOReturn declare_user_activity(void) {
	IOPMAssertionID id;
	return IOPMAssertionDeclareUserActivity(CFSTR("raikirid gamepad input"), kIOPMUserActiveLocal, &id);
}
*/
import "C"

import "fmt"

// declareUserActivity resets the display and system idle timers, the same way
// a keypress on a real HID device would. The dongle is not a HID device for
// macOS, so without this the screen goes to sleep mid-game.
func declareUserActivity() error {
	if r := C.declare_user_activity(); r != C.kIOReturnSuccess {
		return fmt.Errorf("IOPMAssertionDeclareUserActivity: 0x%x", int(r))
	}
	return nil
}

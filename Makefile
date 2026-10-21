BOTTLE ?= Steam
BOTTLES = $(HOME)/Library/Application Support/CrossOver/Bottles
CX_WINE ?= /Applications/CrossOver.app/Contents/SharedSupport/CrossOver/bin/wine
NAMES = xinput1_4 xinput1_3 xinput9_1_0
CFLAGS = -O2 -Wall -Wextra -shared -static-libgcc -Wl,--enable-stdcall-fixup

LABEL = com.shibaev.raikirid
PLIST = $(HOME)/Library/LaunchAgents/$(LABEL).plist

.PHONY: test dll install daemon daemon-uninstall clean

test:
	go vet ./... && go test ./...

dll: build/x64/xinput1_4.dll build/x86/xinput1_4.dll

build/x64/xinput1_4.dll: dll/xinput.c dll/xinput.def
	mkdir -p $(@D)
	x86_64-w64-mingw32-gcc $(CFLAGS) -o $@ $^
	cp $@ $(@D)/xinput1_3.dll && cp $@ $(@D)/xinput9_1_0.dll

build/x86/xinput1_4.dll: dll/xinput.c dll/xinput.def
	mkdir -p $(@D)
	i686-w64-mingw32-gcc $(CFLAGS) -o $@ $^
	cp $@ $(@D)/xinput1_3.dll && cp $@ $(@D)/xinput9_1_0.dll

install: dll
	test -x "$(CX_WINE)" || { echo "CrossOver wine not found at $(CX_WINE), pass CX_WINE=..."; exit 1; }
	cp build/x64/*.dll "$(BOTTLES)/$(BOTTLE)/drive_c/windows/system32/"
	cp build/x86/*.dll "$(BOTTLES)/$(BOTTLE)/drive_c/windows/syswow64/"
	for n in $(NAMES); do \
		"$(CX_WINE)" --bottle "$(BOTTLE)" reg add 'HKCU\Software\Wine\DllOverrides' /v $$n /t REG_SZ /d native /f; \
	done

daemon:
	mkdir -p $(HOME)/.local/bin $(HOME)/Library/LaunchAgents
	launchctl bootout gui/$$(id -u)/$(LABEL) 2>/dev/null || true
	go build -o $(HOME)/.local/bin/raikirid .
	sed "s|@HOME@|$(HOME)|g" raikirid.plist > "$(PLIST)"
	launchctl bootstrap gui/$$(id -u) "$(PLIST)"

daemon-uninstall:
	launchctl bootout gui/$$(id -u)/$(LABEL) 2>/dev/null || true
	rm -f "$(PLIST)" $(HOME)/.local/bin/raikirid

clean:
	rm -rf build

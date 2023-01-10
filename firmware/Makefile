TARGET=pico
TINEGO=tinygo
NAME=$(shell tinygo list .)
include .env

.PHONY: build all flash wait mon

build:
	mkdir -p build
	$(TINYGO) build -target $(TARGET) -o build/$(NAME).elf .

all: flash wait monitor

flash:
	$(TINYGO) flash -target $(TARGET) .

wait:
	sleep 2

mon:
	$(TINYGO) monitor -target $(TARGET)

gdb:
	$(TINYGO) gdb -x -target $(TARGET) -programmer=jlink 

server:
	"C:\Program Files\SEGGER\JLink\JLinkGDBServer.exe" -if swd -port 3333 -speed 4000 -device rp2040_m0_0 &

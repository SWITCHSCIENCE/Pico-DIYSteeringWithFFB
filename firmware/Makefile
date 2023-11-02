TARGET=pico
TINYGO=tinygo
-include .env
NAME=$(notdir $(shell $(TINYGO) list .))

.PHONY: build all flash wait mon

build:
	mkdir -p build
	$(TINYGO) build -target $(TARGET) -o build/$(NAME).uf2 .

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

docker:
	docker compose up build 
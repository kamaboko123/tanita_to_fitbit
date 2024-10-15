# Makefile

TARGET_DIR = bin
TARGET = $(TARGET_DIR)/tanita_to_fitbit

SRC = cmd/main.go cmd/sync.go
SUBMOD = $(wildcard fitbit/*.go) $(wildcard health_planet/*.go)

all: $(TARGET)

$(TARGET): $(SRC) $(SUBMOD)
	go build -o $(TARGET) $(SRC)

clean:
	rm -rf $(TARGET_DIR)

clean-all: clean
	go clean --modcache

run: $(TARGET)
	$(TARGET)


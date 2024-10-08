# Makefile

TARGET_DIR = bin
TARGET = $(TARGET_DIR)/tanita_to_fitbit

SRC = cmd/main.go cmd/sync.go

all: $(TARGET)

$(TARGET): $(SRC)
	go build -o $(TARGET) $(SRC)

clean:
	rm -f $(TARGET)

run: $(TARGET)
	$(TARGET)


.PHONY: build clean test tidy

build:
	go build -o diskimager .

clean:
	rm -f diskimager
	rm -f *.log *.img *.e01 *.bin *.dd
	rm -rf recovered_files_*/

test:
	go test -v ./...

tidy:
	go mod tidy

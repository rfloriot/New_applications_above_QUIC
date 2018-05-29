all: 
	go build -o quicvpn *.go

coverage:
	cd internal; sudo -E go test -coverprofile=coverage.out
	cd internal; go tool cover -html=coverage.out

doc: 
	godoc -goroot ~/go/ quic_vpn


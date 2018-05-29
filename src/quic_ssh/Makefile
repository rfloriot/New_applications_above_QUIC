all: build

build: buildSingle_0_7 buildXXXMulti buildSingle_0_6

buildSingle_0_7:
	go build -o quic_ssh_0_7 *.go

buildXXXMulti: | switchMultiPath compileMulti switchSinglePath

buildSingle_0_6: | switchSingle_0_6 compileSingle_0_6 switchSingle_0_7

compileMulti:
	go build -o quic_ssh_multi *.go

compileSingle_0_6:
	go build -o quic_ssh_0_6 *.go

switchMultiPath:
	cat templates/get_server_cert_multi_path.txt > get_server_cert.go
	mv ../github.com/lucas-clemente/quic-go ../github.com/lucas-clemente/quic-go-single
	mv ../github.com/lucas-clemente/quic-go-multi ../github.com/lucas-clemente/quic-go
	mv ../github.com/bifurcation/mint ../github.com/bifurcation/mint-single
	mv ../github.com/bifurcation/mint-multi ../github.com/bifurcation/mint

switchSinglePath:
	cat templates/get_server_cert_single_path_0_7.txt > get_server_cert.go
	mv ../github.com/lucas-clemente/quic-go ../github.com/lucas-clemente/quic-go-multi
	mv ../github.com/lucas-clemente/quic-go-single ../github.com/lucas-clemente/quic-go
	mv ../github.com/bifurcation/mint ../github.com/bifurcation/mint-multi
	mv ../github.com/bifurcation/mint-single ../github.com/bifurcation/mint

switchSingle_0_6:
	cat templates/get_server_cert_single_path_0_6.txt > get_server_cert.go
	mv ../github.com/lucas-clemente/quic-go ../github.com/lucas-clemente/quic-go-0.7
	mv ../github.com/lucas-clemente/quic-go-0.6 ../github.com/lucas-clemente/quic-go
	mv ../github.com/bifurcation/mint ../github.com/bifurcation/mint-single
	mv ../github.com/bifurcation/mint-multi ../github.com/bifurcation/mint

switchSingle_0_7:
	cat templates/get_server_cert_single_path_0_7.txt > get_server_cert.go
	mv ../github.com/lucas-clemente/quic-go ../github.com/lucas-clemente/quic-go-0.6
	mv ../github.com/lucas-clemente/quic-go-0.7 ../github.com/lucas-clemente/quic-go
	mv ../github.com/bifurcation/mint ../github.com/bifurcation/mint-multi
	mv ../github.com/bifurcation/mint-single ../github.com/bifurcation/mint

runClient: runClientSingle

runClientSingle:
	./quic_ssh_0_6 --pub ../quic_utils/certs/client.pub --priv ../quic_utils/certs/client --req known_hosts_client 127.0.0.1 5050

runClientMulti:
	./quic_ssh_multi --pub ../quic_utils/certs/client.pub --priv ../quic_utils/certs/client --req known_hosts_client 127.0.0.1 5050

runServer: runServerSingle

runServerSingle:
	sudo ./quic_ssh_0_6 -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050

runServerMulti:
	sudo ./quic_ssh_multi -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050

clean:
	rm quic_ssh
	rm quic_ssh_multi
	rm quic_ssh_0_6
	rm quic_ssh_0_7



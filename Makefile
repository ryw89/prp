DESTDIR = /usr/local/bin

prp: main.go
	go get && go build -o prp

install: prp
	install -m 755 prp $(DESTDIR)

DESTDIR = /usr/local/bin

prp: main.go
	go get && go build

install: prp
	install -m 755 prp $(DESTDIR)

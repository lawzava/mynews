build: ensure-dir build-linux compress

ensure-dir:
	rm -rf bin
	mkdir bin

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/mynews.linux-amd64 *.go

compress:
	cd ./bin && find . -name 'mynews*' | xargs -I{} tar czf {}.tar.gz {}

snap-clean:
	rm -f mynews_*_amd64.snap*
	snapcraft clean

snap-build:
	snapcraft

snap-publish:
	snapcraft push --release=edge mynews_*_amd64.snap

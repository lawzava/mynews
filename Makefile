build: ensure-dir build-linux build-windows build-darwin compress

ensure-dir:
	rm -rf bin
	mkdir bin

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/mynews.linux-amd64 cmd/mynews/*.go

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/mynews.windows-amd64.exe cmd/mynews/*.go

build-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/mynews.darwin-amd-64 cmd/mynews/*.go

compress:
	cd ./bin && find . -name 'mynews*' | xargs -I{} tar czf {}.tar.gz {}

snap-clean:
	rm -f mynews_*_amd64.snap*
	snapcraft clean mynews

snap-build:
	snapcraft

snap-install:
	snap install mynews*.snap --dangerous

snap-publish:
	snapcraft push --release=edge mynews_*_amd64.snap

docker-build:
	docker build -t mynews:local .

docker-run:
	docker run mynews:local

compose-run:
	docker-compose up

sudo apt-get update && sudo apt-get install -y snapd git
sudo snap install --classic go && sudo snap install --classic yarn

rm -rf dist/
rm -rf bin/
rm -rf build/
node_modules/.bin/webpack --env prod
WAVESRV_VERSION=$(node -e 'console.log(require("./version.js"))')
WAVESHELL_VERSION=v0.4
GO_LDFLAGS="-s -w -X main.BuildTime=$(date +'%Y%m%d%H%M')"
function buildWaveShell {
    (cd waveshell; CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -ldflags="$GO_LDFLAGS" -o ../bin/mshell/mshell-$WAVESHELL_VERSION-$1.$2 main-waveshell.go)
}
function buildWaveSrv {
    # adds -extldflags=-static, *only* on linux (macos does not support fully static binaries) to avoid a glibc dependency
    (cd wavesrv; CGO_ENABLED=1 GOARCH=$1 go build -tags "osusergo,netgo,sqlite_omit_load_extension" -ldflags "-linkmode 'external' -extldflags=-static $GO_LDFLAGS -X main.WaveVersion=$WAVESRV_VERSION" -o ../bin/wavesrv.$1 ./cmd)
}
buildWaveShell darwin amd64
buildWaveShell darwin arm64
buildWaveShell linux amd64
buildWaveShell linux arm64
buildWaveSrv $GOARCH
yarn run electron-builder -c electron-builder.config.js -l
                      
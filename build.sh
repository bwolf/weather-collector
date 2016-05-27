#!/bin/bash
#
# See https://github.com/influxdata/telegraf/blob/master/CONTRIBUTING.md

readonly go_version=go1.6.2

readonly weather_collector_src=github.com/bwolf/weather-collector
readonly weather_collector_arch=armhf
readonly weather_collector_platform=linux
readonly weather_collector_version=0.0.1
readonly weather_collector_iteration=1
readonly weather_collector_url=http://$weather_collector_src

readonly gvm_url=https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer

echo -- Ensuring gvm
if [ ! -r ~/.gvm/scripts/gvm ]; then
    curl -ogvm-installer -s -S -L $gvm_url
    bash < ~/gvm-installer
fi

# shellcheck disable=SC1090
source ~/.gvm/scripts/gvm || exit 1

echo -- Bootstraping go: $go_version
gvm install go1.4 &&
    gvm use go1.4 &&
    gvm install $go_version &&
    gvm use $go_version --default || exit 1

echo -- Preparing GOPATH=~/go
export GOPATH=~/go
export PATH=$PATH:$GOPATH/bin
mkdir -p $GOPATH

echo -- Building
mkdir -p "$(dirname $GOPATH/src/$weather_collector_src)" &&
    ln -Fsfv /vagrant $GOPATH/src/$weather_collector_src &&
    cd $GOPATH/src/$weather_collector_src &&
    go get -d ./... &&
    cd $GOPATH/src/$weather_collector_src &&
    GOOS=linux GOARM=7 GOARCH=arm go build || exit 1

echo -- Packaging
readonly package_file=weather-collector_${weather_collector_version}-${weather_collector_iteration}_${weather_collector_arch}.deb
readonly package_dir=tmp
rm -f $package_file &&
cd $GOPATH/src/$weather_collector_src &&
    rm -rf $package_dir &&
    mkdir -p $package_dir/bin &&
    mkdir -p $package_dir/lib/weather-collector &&
    cp -a weather-collector $package_dir/bin &&
    cp -a weather-collector.service $package_dir/lib/weather-collector &&
    fpm -s dir -t deb -C $package_dir \
        --prefix /usr \
        --name weather-collector \
        --version $weather_collector_version \
        --iteration $weather_collector_iteration \
        --no-depends \
        --no-auto-depends \
        --architecture $weather_collector_arch \
        --url $weather_collector_url \
        --post-install weather-collector.postinst \
        . &&
    dpkg --contents $package_file || exit 1

# EOF

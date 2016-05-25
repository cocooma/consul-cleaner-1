apk --update add git bash openssh-client
git config --global url."git@github.com:".insteadOf "https://github.com/"
mkdir ~/.ssh
echo -e "Host github.com\n\tStrictHostKeyChecking no\n" >> ~/.ssh/config
go get -d -v github.com/notonthehighstreet/consul-cleaner
go install --ldflags '-extldflags -static -s' -v github.com/notonthehighstreet/consul-cleaner
mkdir /build/build
cp -r /go/bin/* /build/build

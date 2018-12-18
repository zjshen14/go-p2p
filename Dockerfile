FROM golang:1.10.2-stretch

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
		chmod +x /usr/local/bin/dep

COPY ./ $GOPATH/src/github.com/zjshen14/go-p2p/

ARG SKIP_DEP=false

RUN if [ "$SKIP_DEP" != true ] ; \
    then \
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh && \
        cd $GOPATH/src/github.com/zjshen14/go-p2p && \
        	dep ensure -vendor-only; \
    fi

run cd $GOPATH/src/github.com/zjshen14/go-p2p && \
		go build -o ./bin/main -v ./main/main.go

CMD ["/go/src/github.com/zjshen14/go-p2p/bin/main"]

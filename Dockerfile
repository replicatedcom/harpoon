FROM golang:1.6

RUN go get -u github.com/tools/godep

ENV PROJECTPATH=/go/src/github.com/replicatedhq/harpoon
ENV GOPATH $PROJECTPATH/_vendor:$GOPATH

ENV LOG_LEVEL DEBUG

WORKDIR $PROJECTPATH

CMD ["/bin/bash"]

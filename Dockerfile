FROM golang:1.7

RUN go get github.com/kardianos/govendor

ENV PROJECTPATH=/go/src/github.com/replicatedcom/harpoon
ENV PATH $PATH:$PROJECTPATH/go/bin

ENV LOG_LEVEL DEBUG

WORKDIR $PROJECTPATH

CMD ["/bin/bash"]

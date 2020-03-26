FROM golang:1.12

ENV PROJECTPATH=/go/src/github.com/replicatedcom/harpoon
ENV PATH $PATH:$PROJECTPATH/go/bin

ENV LOG_LEVEL DEBUG

WORKDIR $PROJECTPATH

CMD ["/bin/bash"]

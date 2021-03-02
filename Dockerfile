FROM golang:1.16 as builder

WORKDIR /go/src/github.com/dcherman/image-cache-daemon

COPY go.mod go.sum ./
RUN go mod download && go mod tidy

COPY . .
RUN make bin/image-cache-daemon

FROM gcr.io/distroless/static

COPY --from=builder /go/src/github.com/dcherman/image-cache-daemon/bin/image-cache-daemon /image-cache-daemon

ENTRYPOINT [ "/image-cache-daemon" ]
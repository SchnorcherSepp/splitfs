# https://github.com/chemidy/smallest-secured-golang-docker-image/blob/master/docker/distroless_static.Dockerfile
ARG  BUILDER_IMAGE=golang:latest
ARG  DISTROLESS_IMAGE=gcr.io/distroless/static

############################
# STEP 1 build executable binary
############################
FROM ${BUILDER_IMAGE} as builder

# Ensure ca-certficates are up to date
RUN update-ca-certificates

WORKDIR $GOPATH/src/app/

# use modules
COPY go.mod .

ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

COPY . .

# Build the static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -ldflags="-w -s -X main.version=$(git describe --tags) -extldflags '-static'" -a \
      -o /go/bin/app .

############################
# STEP 2 build a small image
############################
# using static nonroot image
# user:group is nobody:nobody, uid:gid = 65534:65534
FROM ${DISTROLESS_IMAGE}

# Copy our static executable
COPY --from=builder /go/bin/app /go/bin/splitfs

# Run the hello binary.
ENTRYPOINT ["/go/bin/splitfs"]
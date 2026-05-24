FROM golang:1.22-alpine AS build

WORKDIR /src

COPY packages/cli/go.mod packages/cli/go.sum ./packages/cli/
WORKDIR /src/packages/cli
RUN go mod download

WORKDIR /src
COPY packages/cli ./packages/cli

WORKDIR /src/packages/cli
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/actlane ./cmd/actlane

FROM alpine:3.20

RUN addgroup -S actlane && adduser -S -G actlane actlane
COPY --from=build /out/actlane /usr/local/bin/actlane

USER actlane
WORKDIR /workspace
ENTRYPOINT ["actlane"]
CMD ["version"]

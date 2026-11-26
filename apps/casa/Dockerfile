# casa: PWA (React) embebida en un binario Go estático.
# Multi-arch (arm64 para las Pi, amd64 para pc1-node) sin emulación: node y go corren en la
# arquitectura del builder ($BUILDPLATFORM) y Go cross-compila con GOARCH=$TARGETARCH.

# ---- 1. Frontend (el resultado es JS/CSS: independiente de la arquitectura)
FROM --platform=$BUILDPLATFORM node:24.11.1-alpine3.22 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- 2. Backend Go con la PWA embebida (go:embed web/dist)
FROM --platform=$BUILDPLATFORM golang:1.25.4-alpine3.22 AS api
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY --from=web /web/dist ./web/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /casa .

# ---- 3. Imagen final: solo el binario (sin shell, usuario nonroot 65532)
# distroless no tiene tags de versión → fijado por digest (regla: nada de "latest")
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=api /casa /casa
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/casa"]

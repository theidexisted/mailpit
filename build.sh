npm install && npm run package
CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/axllent/mailpit/config.Version=1.4_prometheus" -o mailpit

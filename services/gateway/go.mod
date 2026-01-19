module github.com/edalab/services/gateway

go 1.21

require (
	github.com/edalab/pkg/config v0.0.0
	github.com/edalab/pkg/events v0.0.0
	github.com/edalab/pkg/kafka v0.0.0
	github.com/edalab/pkg/observability v0.0.0
	github.com/go-chi/chi/v5 v5.0.11
	github.com/gorilla/websocket v1.5.1
)

replace (
	github.com/edalab/pkg/config => ../../pkg/config
	github.com/edalab/pkg/database => ../../pkg/database
	github.com/edalab/pkg/events => ../../pkg/events
	github.com/edalab/pkg/kafka => ../../pkg/kafka
	github.com/edalab/pkg/observability => ../../pkg/observability
)

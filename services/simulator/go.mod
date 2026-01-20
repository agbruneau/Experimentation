module github.com/edalab/services/simulator

go 1.21

require (
	github.com/edalab/pkg/config v0.0.0
	github.com/edalab/pkg/events v0.0.0
	github.com/edalab/pkg/kafka v0.0.0
	github.com/edalab/pkg/observability v0.0.0
	github.com/go-chi/chi/v5 v5.0.11
	github.com/google/uuid v1.5.0
	github.com/shopspring/decimal v1.3.1
)

replace (
	github.com/edalab/pkg/config => ../../pkg/config
	github.com/edalab/pkg/events => ../../pkg/events
	github.com/edalab/pkg/kafka => ../../pkg/kafka
	github.com/edalab/pkg/observability => ../../pkg/observability
)

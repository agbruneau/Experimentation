module github.com/edalab/tests/integration

go 1.21

require (
	github.com/edalab/pkg/database v0.0.0
	github.com/edalab/pkg/events v0.0.0
	github.com/edalab/pkg/kafka v0.0.0
	github.com/stretchr/testify v1.8.4
	github.com/testcontainers/testcontainers-go v0.27.0
	github.com/testcontainers/testcontainers-go/modules/kafka v0.27.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.27.0
)

replace (
	github.com/edalab/pkg/database => ../../pkg/database
	github.com/edalab/pkg/events => ../../pkg/events
	github.com/edalab/pkg/kafka => ../../pkg/kafka
)

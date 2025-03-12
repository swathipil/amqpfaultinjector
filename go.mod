module github.com/Azure/amqpfaultinjector

go 1.23

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0
	github.com/google/go-cmp v0.6.0
	github.com/joho/godotenv v1.5.1
	github.com/madflojo/testcerts v1.4.0
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	go.uber.org/mock v0.5.0
)

// These are test dependencies, only.
require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.1
	// this is the next beta for azservicebus, which supports using a custom endpoint
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus v1.8.0-beta.2
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-amqp v1.3.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// this is the amqp-high-handle-start commit - it's the same as normal go-amqp, but all handle values start at 200, instead of 0.
replace github.com/Azure/go-amqp v1.3.0 => github.com/richardpark-msft/go-amqp v0.13.10-0.20241205211146-afa568791c13

module faultinjectorsamplego

go 1.23.1

require (
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.8.0

	// this is the next beta for azservicebus, which supports using a custom endpoint
	github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus v1.7.4-0.20241210213713-345bd9830ab3
)

require github.com/joho/godotenv v1.5.1

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.17.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.10.0 // indirect
	github.com/Azure/go-amqp v1.3.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	golang.org/x/crypto v0.35.0 // indirect
	golang.org/x/net v0.36.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

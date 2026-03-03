module wws

go 1.24.0

require (
	github.com/gorilla/mux v1.8.1
	wws/api v0.0.0-00010101000000-000000000000
)

require golang.org/x/oauth2 v0.35.0 // indirect

replace wws/api => ./api

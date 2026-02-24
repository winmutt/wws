module wws

go 1.21

require (
	github.com/gorilla/mux v1.8.1
	wws/api v0.0.0-00010101000000-000000000000
)

replace wws/api => ./api

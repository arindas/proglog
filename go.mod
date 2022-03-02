module github.com/arindas/proglog

replace github.com/arindas/proglog/internal/server => ./internal/server/

go 1.17

require github.com/arindas/proglog/internal/server v0.0.0-00010101000000-000000000000

require github.com/gorilla/mux v1.8.0 // indirect

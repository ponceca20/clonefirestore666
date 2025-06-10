@echo off
echo Ejecutando tests de autenticacion...
go test ./internal/auth/... -v
pause
